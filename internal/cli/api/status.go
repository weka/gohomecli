package api

import (
	"github.com/spf13/cobra"
	"github.com/weka/gohomecli/internal/cli/app"
	"github.com/weka/gohomecli/internal/utils"
	"github.com/weka/gohomecli/pkg/client"
)

func init() {
	app.AppCmd.AddCommand(serverVersionCmd)
	app.AppCmd.AddCommand(dbStatusCmd)
}

var serverVersionCmd = &cobra.Command{
	Use:     "server-version",
	Short:   "Show server version",
	Long:    "Show server version",
	GroupID: "API",
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
	Use:     "db-status",
	Short:   "Show database status",
	Long:    "Show database status",
	GroupID: "API",
	Run: func(cmd *cobra.Command, args []string) {
		client := client.GetClient()
		status, err := client.GetDBStatus()
		if err != nil {
			utils.UserError(err.Error())
		}
		utils.UserOutputJSON(status)
	},
}
