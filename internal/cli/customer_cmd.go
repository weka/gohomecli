package cli

import (
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/weka/gohomecli/internal/client"
	"github.com/weka/gohomecli/internal/utils"
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
			utils.UserError(err.Error())
		}
		utils.RenderTable([]string{"Attribute", "Value"}, func(table *tablewriter.Table) {
			table.Append([]string{"ID", customer.ID})
			table.Append([]string{"Name", customer.Name})
			table.Append([]string{"Monitored", utils.BoolToYesNo(customer.Monitored)})
		})
	},
}

var customerListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all customers",
	Long:  "List all customers",
	Run: func(cmd *cobra.Command, args []string) {
		api := client.GetClient()
		query, err := api.QueryCustomers()
		if err != nil {
			utils.UserError(err.Error())
		}
		utils.RenderTableRows(
			[]string{"ID", "Name", "Monitored"},
			func() []string {
				customer, err := query.NextCustomer()
				if err != nil {
					utils.UserError(err.Error())
				}
				if customer == nil {
					return nil
				}
				return []string{customer.ID, customer.Name, utils.BoolToYesNo(customer.Monitored)}
			})
	},
}
