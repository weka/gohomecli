package cli

import (
	"github.com/spf13/cobra"
	"github.com/weka/gohomecli/internal/env"
	"github.com/weka/gohomecli/internal/utils"
)

var cfgFile string
var siteName string
var verboseLogging bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "homecli",
	Short: "Weka Home Command Line Utility",
	Long:  `Weka Home Command Line Utility`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		utils.UserError(err.Error())
	}
}

func init() {
	cobra.OnInitialize(initLogging)
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().BoolVarP(&verboseLogging, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringVarP(&siteName, "site", "s", "", "target Weka Home site")
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
