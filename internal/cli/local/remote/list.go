package remote

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/weka/gohomecli/internal/local/chart"
	"github.com/weka/gohomecli/internal/utils"
)

type listOptions struct {
	outputFormat string
}

func newListCmd() *cobra.Command {
	opts := &listOptions{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List active remote sessions",
		Long: `List active remote sessions by querying pods with app=remote-session label.

			Examples:
  			   homecli local remote list # List all active sessions in table format
  			   homecli local remote list --output json # List sessions in JSON format
  			   homecli local remote list --output yaml # List sessions in YAML format
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return listRun(cmd, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.outputFormat, "output", "o", "table", "Output format: table, json, yaml")

	return cmd
}

// SessionInfo represents information about an active remote session
type SessionInfo struct {
	SessionID   string `json:"sessionId" yaml:"sessionId"`
	ClusterID   string `json:"clusterId" yaml:"clusterId"`
	ClusterName string `json:"clusterName" yaml:"clusterName"`
	Duration    string `json:"duration" yaml:"duration"`
}

func listRun(cmd *cobra.Command, opts *listOptions) error {
	ctx := cmd.Context()

	k8s, err := chart.NewKubernetesClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// List pods with app=remote-session label
	selector := fmt.Sprintf("%s=%s", remoteAccessLabel, remoteAccessValue)
	pods, err := k8s.Clientset.CoreV1().Pods(chart.ReleaseNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	if len(pods.Items) == 0 {
		utils.UserNote("No active remote sessions")
		return nil
	}

	// Build session info list
	sessions := make([]SessionInfo, 0, len(pods.Items))
	for _, pod := range pods.Items {
		duration := formatDuration(pod.CreationTimestamp.Time)
		sessions = append(sessions, SessionInfo{
			SessionID:   pod.Labels["session-id"],
			ClusterID:   pod.Labels["cluster-id"],
			ClusterName: pod.Labels["cluster-name"],
			Duration:    duration,
		})
	}

	// Output based on format
	switch opts.outputFormat {
	case "json":
		return outputSessionsAsJSON(sessions)
	case "yaml":
		return outputSessionsAsYAML(sessions)
	default:
		return outputSessionsAsTable(sessions)
	}
}

func formatDuration(creationTime time.Time) string {
	duration := time.Since(creationTime)

	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	}
	if duration < time.Hour {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	}
	if duration < 24*time.Hour {
		hours := int(duration.Hours())
		minutes := int(duration.Minutes()) % 60
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
	days := int(duration.Hours()) / 24
	hours := int(duration.Hours()) % 24
	return fmt.Sprintf("%dd%dh", days, hours)
}

func outputSessionsAsJSON(sessions []SessionInfo) error {
	output, err := json.MarshalIndent(sessions, "", "  ")
	if err != nil {
		return err
	}
	utils.UserOutputJSON(output)
	return nil
}

func outputSessionsAsYAML(sessions []SessionInfo) error {
	output, err := yaml.Marshal(sessions)
	if err != nil {
		return err
	}
	fmt.Println(string(output))
	return nil
}

func outputSessionsAsTable(sessions []SessionInfo) error {
	headers := []string{"SESSION", "CLUSTER ID", "CLUSTER NAME", "DURATION"}
	index := 0
	utils.RenderTableRows(headers, func() []string {
		if index < len(sessions) {
			s := sessions[index]
			index++
			// Truncate cluster ID for display
			clusterIDDisplay := s.ClusterID
			if len(clusterIDDisplay) > 36 {
				clusterIDDisplay = clusterIDDisplay[:36]
			}
			return []string{
				s.SessionID,
				clusterIDDisplay,
				s.ClusterName,
				s.Duration,
			}
		}
		return nil
	})
	return nil
}
