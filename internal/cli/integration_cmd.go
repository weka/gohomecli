package cli

import (
	"github.com/spf13/cobra"
	"github.com/weka/gohomecli/internal/utils"
)

func init() {
	rootCmd.AddCommand(integrationCmd)
	integrationCmd.AddCommand(integrationTestCmd)
}

var integrationCmd = &cobra.Command{
	Use:     "integration",
	Aliases: []string{"integrations"}, // backward compatibility
	Short:   "Integrations",
	Long:    "Integrations",
	Run: func(cmd *cobra.Command, args []string) {
		utils.UserError("Not implemented")
	},
}

var integrationTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test integration",
	Long:  "Test integration",
	Run: func(cmd *cobra.Command, args []string) {
		utils.UserError("Not implemented")
	},
}
