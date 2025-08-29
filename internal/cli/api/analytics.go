package api

import (
	"encoding/json"
	"errors"

	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/env"
	"github.com/weka/gohomecli/internal/utils"
	"github.com/weka/gohomecli/pkg/client"
)

func init() {
	Cli.AddHook(func(appCmd *cobra.Command) {
		appCmd.AddCommand(analyticsCmd)
		analyticsCmd.Flags().BoolVarP(&analyticsCmdArgs.allActiveClusters, "all-active", "a",
			false, "get analytics for all active clusters")
		analyticsCmd.Flags().StringVarP(&analyticsCmdArgs.clusterID, "cluster", "c",
			"", "get analytics for this cluster")
	})
}

var analyticsCmdArgs = struct {
	clusterID         string
	allActiveClusters bool
}{}

var analyticsCmd = &cobra.Command{
	Use:     "analytics { --all-active | --cluster ID }",
	Short:   "Get cluster analytics data",
	Long:    "Get cluster analytics data",
	GroupID: "API",
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
			utils.UserError(analyticsCmdArgs.clusterID + " isn't a valid guid")
		}
		if clusterID != "" {
			cluster, err := api.GetCluster(clusterID)
			if err != nil {
				utils.UserError(err.Error())
			}

			analytics := getClusterAnalytics(api, cluster, false)
			outputClusterAnalytics(analytics, false)

			return
		}
		query, err := api.QueryClusters(&client.RequestOptions{Params: client.GetActiveClustersParams()})
		if err != nil {
			utils.UserError(err.Error())
		}
		analyticsList := make([]any, 0)
		for {
			cluster, err := query.NextCluster()
			if err != nil {
				utils.UserError(err.Error())
			}
			if cluster == nil {
				break
			}
			analytics := getClusterAnalytics(api, cluster, true)
			if analytics != nil {
				analyticsList = append(analyticsList, analytics)
			}
		}

		if len(analyticsList) > 0 {
			outputClusterAnalytics(analyticsList, true)
		}
	},
}

var customersCache = make(map[string]string)

func getClusterAnalytics(client *client.Client, cluster *client.Cluster, silenceFailure bool) any {
	analytics, err := client.GetAnalytics(cluster.ID)
	if err != nil {
		if silenceFailure {
			return nil
		}
		utils.UserError("Failed to get analytics for cluster %s: %s", cluster.ID, err)
	}
	customerName := ""
	if cluster.CustomerID != "" {
		if _, ok := customersCache[cluster.CustomerID]; ok {
			customerName = customersCache[cluster.CustomerID]
		} else {
			customer, err := client.GetCustomer(cluster.CustomerID)
			if err != nil {
				if silenceFailure {
					return nil
				}
				utils.UserError("Failed to get customer for cluster %s: %s", cluster.ID, err)
			}
			customersCache[cluster.CustomerID] = customer.Name
			customerName = customer.Name
		}
	}

	var jsn map[string]any
	err = json.Unmarshal(analytics, &jsn)
	if err != nil {
		if silenceFailure {
			return nil
		}
		utils.UserError("Failed to unmarshal analytics json for cluster %s: %s", cluster.ID, err)
	}
	jsn["_meta"] = map[string]string{"customer_name": customerName}

	utils.UserNote("Fetched analytics for cluster %s (%s)", cluster.Name, cluster.ID)
	return jsn
}

func outputClusterAnalytics(jsn any, silenceFailure bool) {
	newAnalytics, err := json.Marshal(jsn)
	if err != nil {
		if silenceFailure {
			return
		}
		utils.UserError("Failed to marshal analytics json: %s", err)
	}
	utils.UserOutputJSON(newAnalytics)
}
