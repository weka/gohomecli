package api

import (
	"strconv"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/utils"
	"github.com/weka/gohomecli/pkg/client"
)

func init() {
	inits = append(inits, func() {
		appCmd.AddCommand(integrationCmd)
		integrationCmd.AddCommand(integrationGetCmd)
		integrationCmd.AddCommand(integrationListCmd)
		integrationCmd.AddCommand(integrationTestCmd)
	})
}

var integrationCmd = &cobra.Command{
	Use:     "integration",
	Short:   "Interact with integrations",
	Long:    "Interact with integrations",
	GroupID: "API",
}

var integrationGetCmd = &cobra.Command{
	Use:   "get <integration-id>",
	Short: "Show a single integration",
	Long:  "Show a single integration",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := client.GetClient()
		integrationID, err := strconv.Atoi(args[0])
		if err != nil {
			utils.UserError("invalid integration ID: %s", args[0])
		}
		integration, err := client.GetIntegration(integrationID)
		if err != nil {
			utils.UserError(err.Error())
		}
		utils.RenderTable([]string{"Attribute", "Value"}, func(table *tablewriter.Table) {
			table.Append([]string{"ID", strconv.Itoa(integration.ID)})
			table.Append([]string{"Name", integration.Name})
			table.Append([]string{"Type", integration.Configuration.Type})
		})
	},
}

var integrationListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all integrations",
	Long:  "List all integrations",
	Run: func(cmd *cobra.Command, args []string) {
		api := client.GetClient()
		query, err := api.QueryIntegrations(nil)
		if err != nil {
			utils.UserError(err.Error())
		}
		utils.RenderTableRows(
			[]string{"ID", "Name", "Type"},
			func() []string {
				integration, err := query.NextIntegration()
				if err != nil {
					utils.UserError(err.Error())
				}
				if integration == nil {
					return nil
				}
				return []string{strconv.Itoa(integration.ID), integration.Name, integration.Configuration.Type}
			})
	},
}

var integrationTestCmd = &cobra.Command{
	Use:   "test <integration-id> <event-code>",
	Short: "Test integration",
	Long:  "Test integration",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		client := client.GetClient()
		integrationID, err := strconv.Atoi(args[0])
		if err != nil {
			utils.UserError("invalid integration ID: %s", args[0])
		}
		//integration, err := client.GetIntegration(integrationID)
		//if err != nil {
		//	utils.UserError(err.Error())
		//}
		eventCode := args[1]
		err = client.TestIntegration(integrationID, eventCode)
		if err != nil {
			utils.UserError("Integration test failed: %s", err)
		}
	},
}
