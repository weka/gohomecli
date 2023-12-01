package app

import (
	"fmt"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/env"
)

// appCmd represents the base command when called without any subcommands
var appCmd = &cobra.Command{
	Use:   "homecli",
	Short: "Weka Home Command Line Utility",
	Long:  `Weka Home Command Line Utility`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		ctx, _ := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGHUP)
		cmd.SetContext(ctx)

		if !env.IsValidColorMode() {
			return fmt.Errorf("invalid color mode: %s", env.ColorMode)
		}

		return nil
	},
	SilenceErrors: true, // we're having custom UserError, so disabling integrated one
	SilenceUsage:  true,
}

func Cmd() *cobra.Command {
	return appCmd
}

func init() {
	cobra.OnInitialize(env.InitEnv)

	appCmd.PersistentFlags().StringVar(&env.SiteName, "site", "",
		"target Weka Home site")
	appCmd.PersistentFlags().BoolVarP(&env.VerboseLogging, "verbose", "v", false,
		"verbose output")
	appCmd.PersistentFlags().StringVar(&env.ColorMode, "color", "auto",
		"colored output, even when stdout is not a terminal")
}
