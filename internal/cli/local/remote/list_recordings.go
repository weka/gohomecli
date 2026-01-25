package remote

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/weka/gohomecli/internal/local/chart"
	"github.com/weka/gohomecli/internal/utils"
)

const (
	sessionRecordingsServerLabel = "app=remote-access-recordings-server"
	httpRequestTimeout           = 10 * time.Second
	fileDownloadTimeout          = 5 * time.Minute

	// dufs path_type values
	dufsPathTypeFile = "File"
	dufsPathTypeDir  = "Dir"
)

type listRecordingsOptions struct {
	clusterID    string
	outputFormat string
}

func newListRecordingsCmd() *cobra.Command {
	opts := &listRecordingsOptions{}

	cmd := &cobra.Command{
		Use:   "list-recordings",
		Short: "List available session recordings",
		Long: `List available session recordings stored in the recordings PVC.

Recordings are stored as asciinema .cast files and can be filtered by cluster ID.

Examples:
  homecli local remote list-recordings
  homecli local remote list-recordings --cluster-id 550e8400-e29b-41d4-a716-446655440000
  homecli local remote list-recordings --output json
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return listRecordingsRun(cmd, opts)
		},
	}

	cmd.Flags().StringVar(&opts.clusterID, "cluster-id", "", "Filter by cluster ID")
	cmd.Flags().StringVarP(&opts.outputFormat, "output", "o", "table", "Output format: table, json, yaml")

	return cmd
}

// RecordingInfo represents information about a recording file
type RecordingInfo struct {
	Filename  string `json:"filename" yaml:"filename"`
	Size      int64  `json:"size" yaml:"size"`
	ModTime   int64  `json:"modTime" yaml:"modTime"`
	ClusterID string `json:"clusterId,omitempty" yaml:"clusterId,omitempty"`
}

// dufsResponse represents the JSON response from dufs directory listing
type dufsResponse struct {
	Paths []dufsEntry `json:"paths"`
}

// dufsEntry represents a file/directory entry from dufs JSON response
type dufsEntry struct {
	Name     string `json:"name"`
	PathType string `json:"path_type"` // "File" or "Dir"
	Size     int64  `json:"size"`
	Mtime    int64  `json:"mtime"` // milliseconds since epoch
}

func listRecordingsRun(cmd *cobra.Command, opts *listRecordingsOptions) error {
	ctx := cmd.Context()

	baseURL, err := chart.GetServiceURL(ctx, sessionRecordingsServerLabel)
	if err != nil {
		return fmt.Errorf("failed to find recordings server: %w", err)
	}

	var recordings []RecordingInfo

	if opts.clusterID != "" {
		recordings, err = listRecordingsForCluster(ctx, baseURL, opts.clusterID)
	} else {
		recordings, err = listAllRecordings(ctx, baseURL)
	}
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

// listRecordingsForCluster lists recordings for a specific cluster ID
func listRecordingsForCluster(ctx context.Context, baseURL, clusterID string) ([]RecordingInfo, error) {
	url := fmt.Sprintf("%s/%s?json", baseURL, clusterID)
	entries, err := fetchDufsDirectory(ctx, url)
	if err != nil {
		// Directory might not exist
		return nil, nil
	}

	return filterAndConvertEntries(entries, clusterID), nil
}

// listAllRecordings lists all recordings from cluster subdirectories and root level
func listAllRecordings(ctx context.Context, baseURL string) ([]RecordingInfo, error) {
	url := fmt.Sprintf("%s?json", baseURL)
	entries, err := fetchDufsDirectory(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to list recordings: %w", err)
	}

	var recordings []RecordingInfo

	for _, entry := range entries {
		if entry.PathType == dufsPathTypeDir {
			// Directory = cluster_id subdirectory
			clusterRecordings, _ := listRecordingsForCluster(ctx, baseURL, entry.Name)
			recordings = append(recordings, clusterRecordings...)
		} else if entry.PathType == dufsPathTypeFile && strings.HasSuffix(entry.Name, ".cast") {
			// Root level .cast file (backward compat - no cluster_id)
			recordings = append(recordings, RecordingInfo{
				Filename:  entry.Name,
				Size:      entry.Size,
				ModTime:   entry.Mtime / 1000, // Convert milliseconds to seconds
				ClusterID: "",
			})
		}
	}

	return recordings, nil
}

// fetchDufsDirectory fetches directory listing from dufs server
func fetchDufsDirectory(ctx context.Context, url string) ([]dufsEntry, error) {
	httpClient := utils.NewHTTPClient()

	body, err := httpClient.Get(ctx, url, httpRequestTimeout)
	if err != nil {
		return nil, err
	}

	var response dufsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return response.Paths, nil
}

// filterAndConvertEntries filters entries by .cast extension
func filterAndConvertEntries(entries []dufsEntry, clusterID string) []RecordingInfo {
	var recordings []RecordingInfo

	for _, entry := range entries {
		if entry.PathType != dufsPathTypeFile || !strings.HasSuffix(entry.Name, ".cast") {
			continue
		}

		recordings = append(recordings, RecordingInfo{
			Filename:  entry.Name,
			Size:      entry.Size,
			ModTime:   entry.Mtime / 1000, // Convert milliseconds to seconds
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
