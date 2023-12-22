package cleanup

import (
	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/cli/app/hooks"
)

var (
	Cli hooks.Cli
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Cleanup local setup",
}

func init() {
	Cli.AddHook(func(appCmd *cobra.Command) {
		appCmd.AddCommand(cleanupCmd)
	})
}
