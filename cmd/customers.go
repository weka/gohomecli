package cmd

import (
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/weka/gohomecli/cli"
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
		customer, err := client.GetCustomer(args[0])
		if err != nil {
			cli.UserError(err.Error())
		}
		cli.RenderTable([]string{"Attribute", "Value"}, func(table *tablewriter.Table) {
			table.Append([]string{"ID", customer.ID})
			table.Append([]string{"Name", customer.Name})
			table.Append([]string{"Monitored", cli.BoolToYesNo(customer.Monitored)})
		})
	},
}

var customerListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all customers",
	Long:  "List all customers",
	Run: func(cmd *cobra.Command, args []string) {
		api := client.GetClient()
		next, err := api.QueryCustomers()
		if err != nil {
			cli.UserError(err.Error())
		}
		header := []string{"ID", "Name", "Version"}
		var customer *client.Customer
		instantiate := func() interface{} {
			customer = &client.Customer{}
			return customer
		}
		getRow := func() []string {
			return []string{customer.ID, customer.Name, cli.BoolToYesNo(customer.Monitored)}
		}
		cli.RenderQueryResults(header, instantiate, next, getRow)
	},
}
