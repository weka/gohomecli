package cmd

import (
	"github.com/spf13/cobra"
	"github.com/weka/gohomecli/cli"
	"strings"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show server version",
	Long:  "Show server version",
	Run: func(cmd *cobra.Command, args []string) {
		cli.UserSuccess(
			"Client version: %s (built on %s)",
			cli.VersionInfo.Name,
			strings.Replace(cli.VersionInfo.BuildTime, "_", " ", -1))
	},
}
