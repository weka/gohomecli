package cli

import (
	"fmt"
	"github.com/spf13/cobra"
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

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "homecli",
	Short: "Weka Home Command Line Utility",
	Long:  `Weka Home Command Line Utility`,
	Args: func(cmd *cobra.Command, args []string) error {
		if !isValidColorMode(colorMode) {
			return fmt.Errorf("invalid color mode: %s", colorMode)
		}
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		utils.UserError(err.Error())
	}
}

func init() {
	cobra.OnInitialize(initEnv)
	cobra.OnInitialize(initLogging)
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&siteName, "site", "",
		"target Weka Home site")
	rootCmd.PersistentFlags().BoolVarP(&verboseLogging, "verbose", "v", false,
		"verbose output")
	rootCmd.PersistentFlags().StringVar(&colorMode, "color", "auto",
		"colored output, even when stdout is not a terminal")
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

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	env.InitConfig(siteName)
}
