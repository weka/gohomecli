package api

import (
	"fmt"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/cli/app"
	"github.com/weka/gohomecli/internal/env"
	"github.com/weka/gohomecli/internal/utils"
	"github.com/weka/gohomecli/pkg/client"
)

func init() {
	app.AppCmd.AddCommand(clusterCmd)
	clusterCmd.AddCommand(clusterGetCmd)
	clusterCmd.AddCommand(clusterListCmd)
	clusterListCmd.Flags().BoolVar(&clusterListCmdArgs.active, "active", false,
		"show only active clusters")
	clusterCmd.AddCommand(clusterAliasCmd)
	clusterListCmd.Flags().IntVar(&clusterListCmdArgs.Limit, "limit", 500,
		"show at most this many clusters")
}

var clusterCmd = &cobra.Command{
	Use:     "cluster",
	Aliases: []string{"clusters"}, // backward compatibility
	Short:   "Interact with clusters",
	Long:    "Interact with clusters",
	GroupID: "API",
}

var clusterGetCmd = &cobra.Command{
	Use:   "get <cluster-id>",
	Short: "Show a single cluster",
	Long:  "Show a single cluster",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := client.GetClient()
		clusterID, err := env.ParseClusterIdentifier(args[0])
		if err != nil {
			utils.UserError(fmt.Sprintf("%s isn't a valid guid", args[0]))
		}
		cluster, err := client.GetCluster(clusterID)
		if err != nil {
			utils.UserError(err.Error())
		}
		var customerName string
		if customer, err := client.GetClusterCustomer(cluster); err == nil {
			customerName = customer.Name
		} else {
			customerName = "N/A"
		}
		utils.RenderTable([]string{"Attribute", "Value"}, func(table *tablewriter.Table) {
			table.Append([]string{"Customer", customerName})
			table.Append([]string{"ID", cluster.ID})
			table.Append([]string{"Name", cluster.Name})
			table.Append([]string{"Version", cluster.Version})
		})
	},
}

var clusterListCmdArgs = struct {
	active bool
	Limit  int
}{}

var clusterListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all clusters",
	Long:  "List all clusters",
	Run: func(cmd *cobra.Command, args []string) {
		api := client.GetClient()
		options := &client.RequestOptions{}
		if clusterListCmdArgs.active {
			options.Params = client.GetActiveClustersParams()
		}
		options.PageSize = clusterListCmdArgs.Limit
		query, err := api.QueryClusters(options)
		if err != nil {
			utils.UserError(err.Error())
		}
		index := 0
		utils.RenderTableRows(
			[]string{"ID", "Name", "Version"},
			func() []string {
				if index >= clusterListCmdArgs.Limit {
					return nil
				}
				cluster, err := query.NextCluster()
				if err != nil {
					utils.UserError(err.Error())
				}
				index++
				if cluster == nil {
					return nil
				}
				return []string{cluster.ID, cluster.Name, cluster.Version}
			})
	},
}
