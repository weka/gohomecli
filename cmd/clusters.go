package cmd

import (
	"context"
	"fmt"
	"github.com/weka/gohomecli/cli"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/weka/gohomecli/cli/client"
)

func init() {
	rootCmd.AddCommand(clusterCmd)
	clusterCmd.AddCommand(clusterGetCmd)
	clusterCmd.AddCommand(clusterListCmd)
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
		ctx := context.Background()
		cluster, err := client.GetCluster(ctx, args[0])
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			os.Exit(2)
		}
		var customerName string
		if customer, err := client.GetClusterCustomer(ctx, cluster); err == nil {
			customerName = customer.Name
		} else {
			customerName = "N/A"
		}
		cli.NewTableRenderer([]string{"Attribute", "Value"}, func(table *tablewriter.Table) {
			table.Append([]string{"Customer", customerName})
			table.Append([]string{"ID", cluster.ID})
			table.Append([]string{"Name", cluster.Name})
			table.Append([]string{"Version", cluster.Version})
		}).Render()
	},
}

var clusterListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all clusters",
	Long:  "List all clusters",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("ALL CLUSTERS")
	},
}
