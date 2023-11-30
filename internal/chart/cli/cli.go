package cli

import (
	"github.com/spf13/cobra"
	"github.com/weka/gohomecli/internal/cli/app"
)

func init() {
	app.AppCmd.AddGroup(&cobra.Group{
		ID:    "chart",
		Title: "Helm chart management",
	})

	chartCmd := &cobra.Command{
		Use:     "chart",
		Short:   "Manage Weka Home Helm chart",
		Long:    `Manage Weka Home Helm chart`,
		GroupID: "chart",
	}

	app.AppCmd.AddCommand(chartCmd)
	chartCmd.AddCommand(buildChartInstallCmd())
}
