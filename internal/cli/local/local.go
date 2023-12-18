package local

import (
	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/cli/app/hooks"
	"github.com/weka/gohomecli/internal/cli/local/dump"
	"github.com/weka/gohomecli/internal/cli/local/setup"
	"github.com/weka/gohomecli/internal/env"
	"github.com/weka/gohomecli/internal/utils"
)

var Cli hooks.Cli

var localGroup = &cobra.Group{
	ID:    "local",
	Title: "LWH Management",
}

var localCmd = &cobra.Command{
	Use:     "local",
	Short:   "Manage local wekahome",
	Long:    "Manage local wekahome",
	GroupID: "local",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if !env.VerboseLogging {
			utils.SetGlobalLoggingLevel(utils.InfoLevel)
		}
	},
}

func init() {
	Cli.AddHook(func(appCmd *cobra.Command) {
		appCmd.AddGroup(localGroup)
		appCmd.AddCommand(localCmd)

		dump.Cli.InitCobra(localCmd)
		setup.Cli.InitCobra(localCmd)
	})
}
