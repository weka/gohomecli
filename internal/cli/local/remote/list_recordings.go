package remote

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/weka/gohomecli/internal/local/chart"
	"github.com/weka/gohomecli/internal/utils"
)

const (
	recordingsSidecarLabel     = "app=remote-access-recordings-sidecar"
	recordingsSidecarContainer = "sidecar"
	recordingsPath             = "/recordings"
	statOutputParts            = 3 // number of parts in stat output: path|size|mtime
)

// ErrInvalidClusterID is returned when a cluster ID is not a valid UUID
var ErrInvalidClusterID = errors.New("invalid cluster ID: must be a valid UUID")

type (
	listRecordingsOptions struct {
		clusterID    string
		outputFormat string
	}

	// RecordingInfo represents information about a recording file
	RecordingInfo struct {
		Filename  string `json:"filename"            yaml:"filename"`
		ClusterID string `json:"clusterId,omitempty" yaml:"clusterId,omitempty"`
		Size      int64  `json:"size"                yaml:"size"`
		ModTime   int64  `json:"modTime"             yaml:"modTime"`
	}
)

func newListRecordingsCmd() *cobra.Command {
	opts := &listRecordingsOptions{}

	cmd := &cobra.Command{
		Use:   "list-recordings",
		Short: "List available session recordings",
		Long: `List available session recordings stored in the recordings PVC.

Recordings are stored as asciinema .cast files and can be filtered by cluster ID.

Examples:
  homecli remote-access list-recordings
  homecli remote-access list-recordings --cluster-id 550e8400-e29b-41d4-a716-446655440000
  homecli remote-access list-recordings --output json
`,
		PreRunE: func(_ *cobra.Command, _ []string) error {
			return validateOutputFormat(opts.outputFormat)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return listRecordingsRun(cmd, opts)
		},
	}

	cmd.Flags().StringVar(&opts.clusterID, "cluster-id", "", "Filter by cluster ID")
	cmd.Flags().StringVarP(&opts.outputFormat, "output", "o", "table", "Output format: table, json, yaml")

	return cmd
}

func listRecordingsRun(cmd *cobra.Command, opts *listRecordingsOptions) error {
	ctx := cmd.Context()

	// Create exec client
	execClient, err := chart.NewK8sExecClient(ctx, recordingsSidecarLabel, recordingsSidecarContainer)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	recordings, err := listRecordings(ctx, execClient, opts.clusterID)
	if err != nil {
		return err
	}

	if len(recordings) == 0 {
		utils.UserNote("No recordings found")

		return nil
	}

	// Output based on format
	switch opts.outputFormat {
	case "json":
		return outputRecordingsAsJSON(recordings)
	case "yaml":
		return outputRecordingsAsYAML(recordings)
	default:
		return outputRecordingsAsTable(recordings)
	}
}

// listRecordings lists recordings, optionally filtered by cluster ID
func listRecordings(ctx context.Context, client *chart.K8sExecClient, clusterID string) ([]RecordingInfo, error) {
	searchPath := recordingsPath
	if clusterID != "" {
		// Validate clusterID is a valid UUID to prevent shell injection
		if _, err := uuid.Parse(clusterID); err != nil {
			return nil, ErrInvalidClusterID
		}
		searchPath = fmt.Sprintf("%s/%s", recordingsPath, clusterID)
	}

	// Check if directory exists first
	_, err := client.Exec(ctx, "test", "-d", searchPath)
	if err != nil {
		// Directory doesn't exist - return empty results (not an error)
		return nil, nil //nolint:nilerr // test -d exits non-zero if dir doesn't exist; that's not an error for us
	}

	// Find all .cast files recursively with stat info
	// Output format: /recordings/[clusterID/]filename.cast|size|mtime
	output, err := client.Exec(ctx,
		"find", searchPath,
		"-name", "*.cast",
		"-type", "f",
		"-exec", "stat", "-c", "%n|%s|%Y", "{}", ";",
	)
	if err != nil {
		// Directory exists but find failed - this is a real error
		return nil, fmt.Errorf("failed to list recordings: %w", err)
	}

	return parseStatOutput(output), nil
}

// parseStatOutput parses the output of stat command into RecordingInfo structs
// Path format: /recordings/filename.cast or /recordings/clusterID/filename.cast
func parseStatOutput(output string) []RecordingInfo {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	recordings := make([]RecordingInfo, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) != statOutputParts {
			continue
		}

		// Extract filename and cluster ID from path
		// /recordings/file.cast -> clusterID="", filename="file.cast"
		// /recordings/abc-123/file.cast -> clusterID="abc-123", filename="file.cast"
		// /recordings/abc-123/subdir/file.cast -> clusterID="abc-123", filename="subdir/file.cast"
		fullPath := parts[0]
		relPath := strings.TrimPrefix(fullPath, recordingsPath+"/")
		pathParts := strings.Split(relPath, "/")

		var filename, clusterID string
		if len(pathParts) == 1 {
			filename = pathParts[0]
		} else {
			clusterID = pathParts[0]
			// Keep the full relative path after cluster ID (handles nested dirs)
			filename = strings.Join(pathParts[1:], "/")
		}

		size, sizeErr := strconv.ParseInt(parts[1], 10, 64)
		mtime, mtimeErr := strconv.ParseInt(parts[2], 10, 64)
		if sizeErr != nil || mtimeErr != nil {
			// Skip malformed entries - log at debug level
			continue
		}

		recordings = append(recordings, RecordingInfo{
			Filename:  filename,
			Size:      size,
			ModTime:   mtime,
			ClusterID: clusterID,
		})
	}

	return recordings
}

func outputRecordingsAsJSON(recordings []RecordingInfo) error {
	output, err := json.MarshalIndent(recordings, "", "  ")
	if err != nil {
		return err
	}
	utils.UserOutputJSON(output)

	return nil
}

func outputRecordingsAsYAML(recordings []RecordingInfo) error {
	output, err := yaml.Marshal(recordings)
	if err != nil {
		return err
	}
	fmt.Println(string(output))

	return nil
}

func outputRecordingsAsTable(recordings []RecordingInfo) error {
	headers := []string{"FILENAME", "SIZE", "MODIFIED", "CLUSTER ID"}
	index := 0
	utils.RenderTableRows(headers, func() []string {
		if index < len(recordings) {
			r := recordings[index]
			index++
			clusterID := r.ClusterID
			if clusterID == "" {
				clusterID = "-"
			}

			return []string{
				r.Filename,
				formatSize(r.Size),
				formatTime(r.ModTime),
				clusterID,
			}
		}

		return nil
	})

	return nil
}

// formatSize formats bytes into human-readable format
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatTime formats unix timestamp into human-readable format
func formatTime(timestamp int64) string {
	t := time.Unix(timestamp, 0)

	return t.Format("2006-01-02 15:04")
}
