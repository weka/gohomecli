package cmd

import (
	"github.com/spf13/cobra"
	"github.com/weka/gohomecli/cli"
	"github.com/weka/gohomecli/cli/client"
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
			cli.UserError(err.Error())
		}
		cli.UserOutput("Server version: %s", status.Version)
	},
}
