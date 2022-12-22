package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/weka/gohomecli/internal/env"
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
		clusterID, err := env.ParseClusterIdentifier(analyticsCmdArgs.clusterID)
		if err != nil {
			utils.UserError(fmt.Sprintf("%s isn't a valid guid", analyticsCmdArgs.clusterID))
		}
		if clusterID != "" {
			cluster, err := api.GetCluster(clusterID)

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

var customersCache = make(map[string]string)

func outputClusterAnalytics(client *client.Client, cluster *client.Cluster, silenceFailure bool) {
	analytics, err := client.GetAnalytics(cluster.ID)
	if err != nil {
		if silenceFailure {
			return
		}
		utils.UserError("Failed to get analytics for cluster %s: %s", cluster.ID, err)
	}
	var customerName string
	if _, ok := customersCache[cluster.CustomerID]; ok {
		customerName = customersCache[cluster.CustomerID]
	} else {
		customer, err := client.GetCustomer(cluster.CustomerID)
		if err != nil {
			if silenceFailure {
				return
			}
			utils.UserError("Failed to get customer for cluster %s: %s", cluster.ID, err)
		}
		customersCache[cluster.CustomerID] = customer.Name
		customerName = customer.Name
	}
	var jsn map[string]interface{}
	err = json.Unmarshal(analytics, &jsn)
	if err != nil {
		if silenceFailure {
			return
		}
		utils.UserError("Failed to unmarshal analytics json for cluster %s: %s", cluster.ID, err)
	}
	jsn["_meta"] = map[string]string{"customer_name": customerName}
	newAnalytics, err := json.Marshal(jsn)
	if err != nil {
		if silenceFailure {
			return
		}
		utils.UserError("Failed to marshal analytics json with customer name for cluster %s: %s", cluster.ID, err)
	}
	utils.UserOutputJSON(newAnalytics)
}
