package remote

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/local/chart"
	"github.com/weka/gohomecli/internal/utils"
)

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
  homecli local remote copy-recording --recording "2024-01-15T10:30:00-abc123.cast" --output /tmp/
  homecli local remote copy-recording --cluster-id "550e8400-e29b-41d4-a716-446655440000" --output /tmp/
  homecli local remote copy-recording --all --output /tmp/
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

	baseURL, err := chart.GetServiceURL(ctx, sessionRecordingsServerLabel)
	if err != nil {
		return fmt.Errorf("failed to find recordings server: %w", err)
	}

	// Determine which files to copy
	var filesToCopy []RecordingInfo

	if opts.recording != "" {
		// Specific file
		filesToCopy = []RecordingInfo{{
			Filename:  opts.recording,
			ClusterID: opts.clusterID,
		}}
	} else {
		var err error
		if opts.clusterID != "" {
			filesToCopy, err = listRecordingsForCluster(ctx, baseURL, opts.clusterID)
		} else {
			// --all case: copy everything when no specific filters provided
			filesToCopy, err = listAllRecordings(ctx, baseURL)
		}
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
		remotePath := recording.Filename
		if recording.ClusterID != "" {
			remotePath = filepath.Join(recording.ClusterID, recording.Filename)
		}

		dstPath := filepath.Join(opts.output, recording.Filename)

		logger.Debug().Str("src", remotePath).Str("dst", dstPath).Msg("Copying file")

		httpClient := utils.NewHTTPClient()
		url := fmt.Sprintf("%s/%s", baseURL, remotePath)
		err := httpClient.DownloadFile(ctx, url, dstPath, fileDownloadTimeout)
		if err != nil {
			utils.UserWarning("Failed to copy %s: %v", recording.Filename, err)
			continue
		}

		utils.UserOutput("  Copied: %s\n", recording.Filename)
		copiedCount++
	}

	utils.UserNote("Successfully copied %d/%d recording(s)", copiedCount, len(filesToCopy))
	return nil
}
