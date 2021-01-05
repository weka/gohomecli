package cli

import (
	"github.com/hokaccha/go-prettyjson"
	"github.com/spf13/cobra"
	"github.com/weka/gohomecli/internal/client"
	"github.com/weka/gohomecli/internal/utils"
)

func init() {
	rootCmd.AddCommand(serverVersionCmd)
	rootCmd.AddCommand(dbStatusCmd)
}

var serverVersionCmd = &cobra.Command{
	Use:   "server-version",
	Short: "Show server version",
	Long:  "Show server version",
	Run: func(cmd *cobra.Command, args []string) {
		client := client.GetClient()
		status, err := client.GetServerStatus()
		if err != nil {
			utils.UserError(err.Error())
		}
		utils.UserOutput("Server version: %s", status.Version)
	},
}

var dbStatusCmd = &cobra.Command{
	Use:   "db-status",
	Short: "Show database status",
	Long:  "Show database status",
	Run: func(cmd *cobra.Command, args []string) {
		client := client.GetClient()
		status, err := client.GetDBStatus()
		if err != nil {
			utils.UserError(err.Error())
		}
		formatted, err := prettyjson.Format(status)
		if err != nil {
			utils.UserWarning("Failed to colorize JSON: %s", err)
			utils.UserOutput(string(status))
		} else {
			utils.UserOutput(string(formatted))
		}
	},
}
