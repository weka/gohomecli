package cli

import (
	"fmt"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/cli/api"
	"github.com/weka/gohomecli/internal/cli/config"
	"github.com/weka/gohomecli/internal/cli/k3s"
	"github.com/weka/gohomecli/internal/env"
	"github.com/weka/gohomecli/internal/utils"
)

var cfgFile string
var siteName string
var verboseLogging bool
var colorMode string

func isValidColorMode(mode string) bool {
	for _, m := range []string{"auto", "always", "never"} {
		if mode == m {
			return true
		}
	}
	return false
}

// appCmd represents the base command when called without any subcommands
var appCmd = &cobra.Command{
	Use:   "homecli",
	Short: "Weka Home Command Line Utility",
	Long:  `Weka Home Command Line Utility`,
	Args: func(cmd *cobra.Command, args []string) error {
		if !isValidColorMode(colorMode) {
			return fmt.Errorf("invalid color mode: %s", colorMode)
		}
		return nil
	},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		ctx, _ := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGHUP)
		cmd.SetContext(ctx)
	},
	SilenceErrors: true,
	SilenceUsage:  true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := appCmd.Execute(); err != nil {
		utils.UserError(err.Error())
	}
}

func init() {
	cobra.OnInitialize(initEnv, initLogging, initConfig)

	appCmd.PersistentFlags().StringVar(&siteName, "site", "",
		"target Weka Home site")
	appCmd.PersistentFlags().BoolVarP(&verboseLogging, "verbose", "v", false,
		"verbose output")
	appCmd.PersistentFlags().StringVar(&colorMode, "color", "auto",
		"colored output, even when stdout is not a terminal")

	api.Init(appCmd)
	config.Init(appCmd)
	k3s.Init(appCmd)
}

func initEnv() {
	switch colorMode {
	case "always":
		utils.IsColorOutputSupported = true
	case "never":
		utils.IsColorOutputSupported = false
	case "auto":
		utils.IsColorOutputSupported = env.IsInteractiveTerminal
	}
}

func initLogging() {
	if verboseLogging {
		utils.SetGlobalLoggingLevel(utils.DebugLevel)
	}
}

func initConfig() {
	env.InitConfig(siteName)
}
