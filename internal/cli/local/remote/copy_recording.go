package remote

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/local/chart"
	"github.com/weka/gohomecli/internal/utils"
)

// safeFilenamePattern allows safe characters for recording filenames
var safeFilenamePattern = regexp.MustCompile(`^[a-zA-Z0-9_\-:.]+\.cast$`)

type copyRecordingOptions struct {
	recording string
	clusterID string
	all       bool
	output    string
}

func newCopyRecordingCmd() *cobra.Command {
	opts := &copyRecordingOptions{}

	cmd := &cobra.Command{
		Use:   "copy-recording",
		Short: "Copy recording files from PVC to local filesystem",
		Long: `Copy session recording files from the recordings PVC to a local directory.

Recordings can be copied individually or in bulk by cluster ID or all at once.

Examples:
  homecli remote-access copy-recording --recording "2024-01-15T10:30:00-abc123.cast" --output /tmp/
  homecli remote-access copy-recording --cluster-id "550e8400-e29b-41d4-a716-446655440000" --output /tmp/
  homecli remote-access copy-recording --all --output /tmp/
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return copyRecordingRun(cmd, opts)
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if opts.recording == "" && opts.clusterID == "" && !opts.all {
				return fmt.Errorf("must specify one of: --recording, --cluster-id, or --all")
			}
			if opts.output == "" {
				return fmt.Errorf("--output is required")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.recording, "recording", "", "Specific recording filename to copy")
	cmd.Flags().StringVar(&opts.clusterID, "cluster-id", "", "Copy all recordings for a cluster")
	cmd.Flags().BoolVar(&opts.all, "all", false, "Copy all recordings")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "Local destination directory (required)")
	_ = cmd.MarkFlagRequired("output")

	return cmd
}

func copyRecordingRun(cmd *cobra.Command, opts *copyRecordingOptions) error {
	ctx := cmd.Context()

	// Ensure output directory exists
	if err := os.MkdirAll(opts.output, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create exec client
	execClient, err := chart.NewK8sExecClient(ctx, recordingsSidecarLabel, recordingsSidecarContainer)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Determine which files to copy
	var filesToCopy []RecordingInfo

	if opts.recording != "" {
		// Validate inputs to prevent path traversal
		if !safeFilenamePattern.MatchString(opts.recording) {
			return fmt.Errorf("invalid recording filename: must be a .cast file with safe characters")
		}
		if opts.clusterID != "" {
			if _, parseErr := uuid.Parse(opts.clusterID); parseErr != nil {
				return fmt.Errorf("invalid cluster ID: must be a valid UUID")
			}
			// Cluster ID explicitly provided
			filesToCopy = []RecordingInfo{{
				Filename:  opts.recording,
				ClusterID: opts.clusterID,
			}}
		} else {
			// No cluster ID provided - search for the recording to find its cluster ID
			allRecordings, listErr := listRecordings(ctx, execClient, "")
			if listErr != nil {
				return fmt.Errorf("failed to list recordings: %w", listErr)
			}
			for _, r := range allRecordings {
				if r.Filename == opts.recording {
					filesToCopy = []RecordingInfo{r}
					break
				}
			}
			if len(filesToCopy) == 0 {
				return fmt.Errorf("recording not found: %s", opts.recording)
			}
		}
	} else {
		// List by cluster ID or all (empty clusterID = all)
		// listRecordings validates clusterID internally
		filesToCopy, err = listRecordings(ctx, execClient, opts.clusterID)
		if err != nil {
			return fmt.Errorf("failed to list recordings: %w", err)
		}
	}

	if len(filesToCopy) == 0 {
		utils.UserNote("No matching recordings found")
		return nil
	}

	utils.UserOutput("Copying %d recording(s) to %s\n", len(filesToCopy), opts.output)

	// Copy each file
	copiedCount := 0
	for _, recording := range filesToCopy {
		// Construct remote path from ClusterID + Filename
		// Filename may contain subdirectories (e.g., "subdir/file.cast")
		remotePath := filepath.Join(recordingsPath, recording.Filename)
		if recording.ClusterID != "" {
			remotePath = filepath.Join(recordingsPath, recording.ClusterID, recording.Filename)
		}

		// Use basename for local destination (flatten directory structure)
		localFilename := filepath.Base(recording.Filename)
		dstPath := filepath.Join(opts.output, localFilename)

		if err := execClient.CopyFromPod(ctx, remotePath, dstPath); err != nil {
			utils.UserWarning("Failed to copy %s: %v", localFilename, err)
			continue
		}

		utils.UserOutput("  Copied: %s\n", localFilename)
		copiedCount++
	}

	utils.UserNote("Successfully copied %d/%d recording(s)", copiedCount, len(filesToCopy))
	return nil
}
