package remote

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/local/chart"
	"github.com/weka/gohomecli/internal/utils"
)

const outputDirPerms = 0o750

var (
	// safeFilenamePattern allows safe characters for recording filenames
	safeFilenamePattern = regexp.MustCompile(`^[a-zA-Z0-9_\-:.]+\.cast$`)

	// ErrMissingCopyFilter is returned when no filter is specified for copy-recording command
	ErrMissingCopyFilter = errors.New("must specify one of: --recording, --cluster-id, or --all")

	// ErrInvalidFilename is returned when a recording filename contains unsafe characters
	ErrInvalidFilename = errors.New("invalid recording filename: must be a .cast file with safe characters")
)

type (
	copyRecordingOptions struct {
		recording string
		clusterID string
		output    string
		all       bool
	}

	// recordingNotFoundError is returned when a specific recording cannot be found.
	recordingNotFoundError struct {
		name string
	}
)

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
		RunE: func(cmd *cobra.Command, _ []string) error {
			return copyRecordingRun(cmd, opts)
		},
		PreRunE: func(_ *cobra.Command, _ []string) error {
			if opts.recording == "" && opts.clusterID == "" && !opts.all {
				return ErrMissingCopyFilter
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&opts.recording, "recording", "", "Specific recording filename to copy")
	cmd.Flags().StringVar(&opts.clusterID, "cluster-id", "", "Copy all recordings for a cluster")
	cmd.Flags().BoolVar(&opts.all, "all", false, "Copy all recordings")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "Local destination directory (required)")
	//nolint:errcheck,gosec // cobra returns error only for unknown flags, which won't happen with our flag
	cmd.MarkFlagRequired("output")

	return cmd
}

func copyRecordingRun(cmd *cobra.Command, opts *copyRecordingOptions) error {
	ctx := cmd.Context()

	// Ensure output directory exists
	if err := os.MkdirAll(opts.output, outputDirPerms); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create exec client
	execClient, err := chart.NewK8sExecClient(ctx, recordingsSidecarLabel, recordingsSidecarContainer)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Determine which files to copy
	filesToCopy, err := resolveFilesToCopy(ctx, execClient, opts)
	if err != nil {
		return err
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

// resolveFilesToCopy determines which recording files to copy based on options.
func resolveFilesToCopy(
	ctx context.Context,
	execClient *chart.K8sExecClient,
	opts *copyRecordingOptions,
) ([]RecordingInfo, error) {
	if opts.recording == "" {
		// List by cluster ID or all (empty clusterID = all)
		// listRecordings validates clusterID internally
		return listRecordings(ctx, execClient, opts.clusterID)
	}

	return resolveSpecificRecording(ctx, execClient, opts)
}

// resolveSpecificRecording finds a specific recording by name.
func resolveSpecificRecording(
	ctx context.Context,
	execClient *chart.K8sExecClient,
	opts *copyRecordingOptions,
) ([]RecordingInfo, error) {
	// Validate inputs to prevent path traversal
	if !safeFilenamePattern.MatchString(opts.recording) {
		return nil, ErrInvalidFilename
	}

	if opts.clusterID != "" {
		return resolveWithClusterID(opts)
	}

	return searchForRecording(ctx, execClient, opts.recording)
}

// resolveWithClusterID returns recording info when cluster ID is explicitly provided.
func resolveWithClusterID(opts *copyRecordingOptions) ([]RecordingInfo, error) {
	if _, parseErr := uuid.Parse(opts.clusterID); parseErr != nil {
		return nil, ErrInvalidClusterID
	}

	return []RecordingInfo{{
		Filename:  opts.recording,
		ClusterID: opts.clusterID,
	}}, nil
}

// searchForRecording searches all recordings to find the one matching the filename.
func searchForRecording(
	ctx context.Context,
	execClient *chart.K8sExecClient,
	recordingName string,
) ([]RecordingInfo, error) {
	allRecordings, listErr := listRecordings(ctx, execClient, "")
	if listErr != nil {
		return nil, fmt.Errorf("failed to list recordings: %w", listErr)
	}

	for _, r := range allRecordings {
		if r.Filename == recordingName {
			return []RecordingInfo{r}, nil
		}
	}

	return nil, fmt.Errorf("recording not found: %w", &recordingNotFoundError{name: recordingName})
}

func (e *recordingNotFoundError) Error() string {
	return e.name
}
