package app

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/env"
	"github.com/weka/gohomecli/internal/utils"
)

func init() {
	AppCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show client version",
	Long:  "Show client version",
	Run: func(cmd *cobra.Command, args []string) {
		utils.UserOutput(
			"Client version: %s (built on %s)",
			env.VersionInfo.Name,
			strings.Replace(env.VersionInfo.BuildTime, "_", " ", -1))
	},
}
