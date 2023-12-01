package chart

import (
	"github.com/spf13/cobra"
	"github.com/weka/gohomecli/internal/cli/app/hooks"
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
		}

		appCmd.AddCommand(chartCmd)
		chartCmd.AddCommand(buildChartInstallCmd())
	})
}
