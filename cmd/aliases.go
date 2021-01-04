package cmd

import (
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/cli"
)

func init() {
	aliasCmd.AddCommand(aliasListCmd)
	aliasCmd.AddCommand(aliasAddCmd)
	aliasCmd.AddCommand(configSiteUpdateCmd)
	aliasCmd.AddCommand(aliasRemoveCmd)
}

var aliasCmd = &cobra.Command{
	Use:   "alias",
	Short: "Alias commands",
	Long:  "Alias commands",
}

var aliasListCmd = &cobra.Command{
	Use:   "list",
	Short: "List aliases",
	Long:  "List aliases",
	Run: func(cmd *cobra.Command, args []string) {
		aliases := cli.NewAliases()
		cli.RenderTable([]string{"Alias", "Cluster ID"}, func(table *tablewriter.Table) {
			aliases.Iter(func(alias string, clusterID string) {
				table.Append([]string{alias, clusterID})
			})
		})
	},
}

var aliasAddCmd = &cobra.Command{
	Use:   "add <alias> <cluster-id>",
	Short: "Add an alias",
	Long:  "Add an alias",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		alias, clusterID := args[0], args[1]
		aliases := cli.NewAliases()
		existingClusterID, aliasExists := aliases.Get(alias)
		if aliasExists {
			if existingClusterID == clusterID {
				cli.UserWarning("Alias \"%s\" already exists for cluster ID %s", alias, clusterID)
				return
			}
			cli.UserError("Alias \"%s\" already exists for another cluster ID: %s", alias, clusterID)
		}
		err := aliases.Set(alias, clusterID, true)
		if err != nil {
			cli.UserError("Failed to set alias: %s", err)
		}
		cli.UserSuccess("Added alias \"%s\" for cluster ID %s", alias, clusterID)
	},
}

var aliasRemoveCmd = &cobra.Command{
	Use:   "remove <alias>",
	Short: "Remove an alias",
	Long:  "Remove an alias",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		alias := args[0]
		aliases := cli.NewAliases()
		clusterID, aliasExists := aliases.Get(alias)
		if !aliasExists {
			cli.UserError("No such alias: \"%s\"", alias)
		}
		err := aliases.Remove(alias)
		if err != nil {
			cli.UserError("Failed to remove alias: %s", err)
		}
		cli.UserSuccess("Removed alias \"%s\" for cluster ID %s", alias, clusterID)
	},
}
