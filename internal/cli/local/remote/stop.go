package remote

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/weka/gohomecli/internal/local/chart"
	"github.com/weka/gohomecli/internal/utils"
)

// sessionIDPattern matches 6 hex characters (format from generateShortID)
var sessionIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{6}$`)

// ErrMissingStopFilter is returned when no filter is specified for stop command
var ErrMissingStopFilter = errors.New("must specify one of: --session-id, --cluster-id, or --all")

type stopOptions struct {
	sessionID string
	clusterID string
	all       bool
}

func newStopCmd() *cobra.Command {
	opts := &stopOptions{}

	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop running remote session(s)",
		Long: `Stop running remote session(s).


Examples:
 homecli remote-access stop --session-id a1b2c3 # Stop a specific session by ID
 homecli remote-access stop --cluster-id 550e8400-e29b-41d4-a716-446655440000 # Stop all sessions for a cluster
 homecli remote-access stop --all # Stop all active sessions
		`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return stopRun(cmd, opts)
		},
		PreRunE: func(_ *cobra.Command, _ []string) error {
			if opts.sessionID == "" && opts.clusterID == "" && !opts.all {
				return ErrMissingStopFilter
			}

			// Validate session-id format to prevent label selector injection
			if opts.sessionID != "" && !sessionIDPattern.MatchString(opts.sessionID) {
				return errors.New("invalid session-id format: must be 6 hex characters (e.g., a1b2c3)")
			}

			// Validate cluster-id format to prevent label selector injection
			if opts.clusterID != "" {
				if _, err := uuid.Parse(opts.clusterID); err != nil {
					return errors.New("invalid cluster-id format: must be a valid UUID")
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&opts.sessionID, "session-id", "", "Stop specific session by ID")
	cmd.Flags().StringVar(&opts.clusterID, "cluster-id", "", "Stop all sessions for a cluster")
	cmd.Flags().BoolVar(&opts.all, "all", false, "Stop all active sessions")

	return cmd
}

func stopRun(cmd *cobra.Command, opts *stopOptions) error {
	ctx := cmd.Context()

	k8s, err := chart.NewKubernetesClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Build label selector
	selector := fmt.Sprintf("%s=%s", remoteAccessLabel, remoteAccessValue)
	if opts.sessionID != "" {
		selector += ",session-id=" + opts.sessionID
	} else if opts.clusterID != "" {
		selector += ",cluster-id=" + opts.clusterID
	}
	// --all: no additional filter

	logger.Debug().Str("selector", selector).Msg("Listing pods to stop")

	// List matching pods
	pods, err := k8s.Clientset.CoreV1().Pods(chart.ReleaseNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	if len(pods.Items) == 0 {
		utils.UserNote("No matching sessions found")

		return nil
	}

	// Delete each pod
	stoppedCount := 0
	for i := range pods.Items {
		pod := &pods.Items[i]
		sessionID := pod.Labels["session-id"]
		logger.Info().Str("pod", pod.Name).Str("sessionID", sessionID).Msg("Stopping session pod")

		if err := k8s.Clientset.CoreV1().Pods(chart.ReleaseNamespace).Delete(ctx, pod.Name, metav1.DeleteOptions{}); err != nil {
			utils.UserWarning("Failed to stop session %s: %v", sessionID, err)
		} else {
			utils.UserOutput("Stopped session: %s (pod: %s)\n", sessionID, pod.Name)
			stoppedCount++
		}
	}

	utils.UserNote("Stopped %d session(s)", stoppedCount)

	return nil
}
