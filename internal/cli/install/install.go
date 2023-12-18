package install

import (
	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/cli/app/hooks"
	"github.com/weka/gohomecli/internal/cli/install/chart"
	"github.com/weka/gohomecli/internal/cli/install/configure"
	"github.com/weka/gohomecli/internal/cli/install/k3s"
	"github.com/weka/gohomecli/internal/env"
	"github.com/weka/gohomecli/internal/utils"
)

var Cli hooks.Cli

var installGroup = &cobra.Group{
	ID:    "install",
	Title: "Manage installations",
}

var installCmd = &cobra.Command{
	Use:     "install",
	Short:   "Manage installations",
	Long:    "Manage wekahome installations",
	GroupID: "install",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if !env.VerboseLogging {
			utils.SetGlobalLoggingLevel(utils.InfoLevel)
		}
	},
}

func init() {
	Cli.AddHook(func(appCmd *cobra.Command) {
		appCmd.AddGroup(installGroup)
		appCmd.AddCommand(installCmd)
		k3s.Cli.InitCobra(installCmd)
		chart.Cli.InitCobra(installCmd)
		configure.Cli.InitCobra(installCmd)
	})
}
