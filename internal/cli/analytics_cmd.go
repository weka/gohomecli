package cli

import (
	"errors"
	"github.com/spf13/cobra"
	"github.com/weka/gohomecli/internal/utils"
	"github.com/weka/gohomecli/pkg/client"
)

var analyticsCmdArgs = struct {
	allActiveClusters bool
	clusterID         string
}{}

func init() {
	rootCmd.AddCommand(analyticsCmd)
	analyticsCmd.Flags().BoolVarP(&analyticsCmdArgs.allActiveClusters, "all-active", "a",
		false, "get analytics for all active clusters")
	analyticsCmd.Flags().StringVarP(&analyticsCmdArgs.clusterID, "cluster", "c",
		"", "get analytics for this cluster")
}

var analyticsCmd = &cobra.Command{
	Use:   "analytics { --all-active | --cluster ID }",
	Short: "Get cluster analytics data",
	Long:  "Get cluster analytics data",
	Args: func(cmd *cobra.Command, args []string) error {
		if !analyticsCmdArgs.allActiveClusters && analyticsCmdArgs.clusterID == "" {
			return errors.New("please specify either --all-active or --cluster")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		api := client.GetClient()
		if analyticsCmdArgs.clusterID != "" {
			cluster, err := api.GetCluster(analyticsCmdArgs.clusterID)
			if err != nil {
				utils.UserError(err.Error())
			}
			outputClusterAnalytics(api, cluster, false)
			return
		}
		query, err := api.QueryClusters(&client.RequestOptions{Params: client.GetActiveClustersParams()})
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
			outputClusterAnalytics(api, cluster, true)
		}
	},
}

func outputClusterAnalytics(client *client.Client, cluster *client.Cluster, silenceFailure bool) {
	analytics, err := client.GetAnalytics(cluster.ID)
	if err != nil {
		if silenceFailure {
			return
		}
		utils.UserError("Failed to get analytics for cluster %s: %s", cluster.ID, err)
	}
	utils.UserOutputJSON(analytics)
}
