package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"home.weka.io/cli"
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
	Use:   "get",
	Short: "Show a single cluster",
	Long:  "Show a single cluster",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("requires exactly 1 argument (cluster ID)")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		client := cli.GetClient()
		ctx := context.Background()
		cluster, err := client.GetCluster(ctx, args[0])
		if err != nil {
			fmt.Printf("Error: %s", err)
			os.Exit(2)
		}
		fmt.Printf("%v\n", cluster)
	},
}

var clusterListCmd = &cobra.Command{
	Use:   "get",
	Short: "List all clusters",
	Long:  "List all clusters",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("ALL CLUSTERS")
	},
}
