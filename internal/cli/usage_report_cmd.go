package cli

import (
	"errors"
	"github.com/spf13/cobra"
	"github.com/weka/gohomecli/internal/client"
	"github.com/weka/gohomecli/internal/utils"
)

var usageReportCmdArgs = struct {
	allActiveClusters bool
	clusterID         string
}{}

func init() {
	rootCmd.AddCommand(usageReportCmd)
	usageReportCmd.Flags().BoolVarP(&usageReportCmdArgs.allActiveClusters, "all-active", "a",
		false, "get usage report for all active clusters")
	usageReportCmd.Flags().StringVarP(&usageReportCmdArgs.clusterID, "cluster", "c",
		"", "get usage report for this cluster")
}

var usageReportCmd = &cobra.Command{
	Use:   "usage-report { --all-active | --cluster ID }",
	Short: "Get cluster usage report",
	Long:  "Get cluster usage report",
	Args: func(cmd *cobra.Command, args []string) error {
		if !usageReportCmdArgs.allActiveClusters && usageReportCmdArgs.clusterID == "" {
			return errors.New("please specify either --all-active or --cluster")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		client := client.GetClient()
		if usageReportCmdArgs.clusterID != "" {
			cluster, err := client.GetCluster(usageReportCmdArgs.clusterID)
			if err != nil {
				utils.UserError(err.Error())
			}
			outputClusterUsageReport(client, cluster, false)
			return
		}
		query, err := client.QueryClusters()
		if err != nil {
			utils.UserError(err.Error())
		}
		for {
			cluster, err := query.NextCluster()
			if err != nil {
				utils.UserError(err.Error())
			}
			if cluster == nil {
				break
			}
			outputClusterUsageReport(client, cluster, true)
		}
	},
}

func outputClusterUsageReport(client *client.Client, cluster *client.Cluster, silenceFailure bool) {
	report, err := client.GetUsageReport(cluster.ID)
	if err != nil {
		if silenceFailure {
			return
		}
		utils.UserError("Failed to get usage report for cluster %s: %s", cluster.ID, err)
	}
	utils.UserOutputJSON(report)
}
