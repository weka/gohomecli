package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/weka/gohomecli/cli/client"
)

func init() {
	rootCmd.AddCommand(customerCmd)
	customerCmd.AddCommand(customerGetCmd)
	customerCmd.AddCommand(customerListCmd)
}

var customerCmd = &cobra.Command{
	Use:   "customer",
	Short: "Interact with customers",
	Long:  "Interact with customers",
}

var customerGetCmd = &cobra.Command{
	Use:   "get <customer-id>",
	Short: "Show a single customer",
	Long:  "Show a single customer",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := client.GetClient()
		ctx := context.Background()
		customer, err := client.GetCustomer(ctx, args[0])
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			os.Exit(2)
		}
		table := tablewriter.NewWriter(os.Stdout)
		table.SetBorder(false)
		table.SetHeader([]string{"Attribute", "Value"})
		table.Append([]string{"ID", customer.ID})
		table.Append([]string{"Name", customer.Name})
		table.Append([]string{"Monitored", BoolToYesNo(customer.Monitored)})
		table.Render()
	},
}

func BoolToYesNo(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

var customerListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all customers",
	Long:  "List all customers",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("ALL CLUSTERS")
	},
}
