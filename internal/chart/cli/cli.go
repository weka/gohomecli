package cli

import (
	"os/signal"
	"syscall"

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
		PreRun: func(cmd *cobra.Command, args []string) {
			ctx, _ := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGHUP)
			cmd.SetContext(ctx)
		},
	}

	app.AppCmd.AddCommand(chartCmd)
	chartCmd.AddCommand(buildChartInstallCmd())
}
