package cmd

import (
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/weka/gohomecli/cli"
	"github.com/weka/gohomecli/cli/client"
)

func init() {
	rootCmd.AddCommand(clusterCmd)
	clusterCmd.AddCommand(clusterGetCmd)
	clusterCmd.AddCommand(clusterListCmd)
	clusterCmd.AddCommand(aliasCmd)
}

var clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Interact with clusters",
	Long:  "Interact with clusters",
}

var clusterGetCmd = &cobra.Command{
	Use:   "get <cluster-id>",
	Short: "Show a single cluster",
	Long:  "Show a single cluster",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := client.GetClient()
		cluster, err := client.GetCluster(args[0])
		if err != nil {
			cli.UserError(err.Error())
		}
		var customerName string
		if customer, err := client.GetClusterCustomer(cluster); err == nil {
			customerName = customer.Name
		} else {
			customerName = "N/A"
		}
		cli.RenderTable([]string{"Attribute", "Value"}, func(table *tablewriter.Table) {
			table.Append([]string{"Customer", customerName})
			table.Append([]string{"ID", cluster.ID})
			table.Append([]string{"Name", cluster.Name})
			table.Append([]string{"Version", cluster.Version})
		})
	},
}

var clusterListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all clusters",
	Long:  "List all clusters",
	Run: func(cmd *cobra.Command, args []string) {
		api := client.GetClient()
		query, err := api.QueryClusters()
		if err != nil {
			cli.UserError(err.Error())
		}
		cli.RenderTableRows(
			[]string{"ID", "Name", "Version"},
			func() []string {
				cluster, err := query.NextCluster()
				if err != nil {
					cli.UserError("Failed to get next cluster: %s", err)
				}
				if cluster == nil {
					return nil
				}
				return []string{cluster.ID, cluster.Name, cluster.Version}
			})
	},
}
