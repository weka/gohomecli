package chart

import (
	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/cli/app/hooks"
	"github.com/weka/gohomecli/internal/env"
	"github.com/weka/gohomecli/internal/utils"
)

var Cli hooks.Cli

func init() {
	Cli.AddHook(func(appCmd *cobra.Command) {
		appCmd.AddGroup(&cobra.Group{
			ID:    "chart",
			Title: "Helm chart management",
		})

		chartCmd := &cobra.Command{
			Use:     "chart",
			Short:   "Manage Weka Home Helm chart",
			Long:    `Manage Weka Home Helm chart`,
			GroupID: "chart",
			PersistentPreRun: func(cmd *cobra.Command, args []string) {
				if !env.VerboseLogging {
					utils.SetGlobalLoggingLevel(utils.InfoLevel)
				}
			},
		}

		appCmd.AddCommand(chartCmd)
		chartCmd.AddCommand(buildChartInstallCmd())
	})
}
