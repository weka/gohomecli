package cli

import (
	"github.com/spf13/cobra"
	"github.com/weka/gohomecli/internal/client"
	"github.com/weka/gohomecli/internal/utils"
)

func init() {
	rootCmd.AddCommand(serverVersionCmd)
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
