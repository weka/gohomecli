package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/weka/gohomecli/cli"
	"github.com/weka/gohomecli/cli/logging"
	"os"
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
		fmt.Println(err)
		os.Exit(1)
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
		logging.SetGlobalLoggingLevel(logging.DebugLevel)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	cli.InitConfig(siteName)
}
