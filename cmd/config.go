package cmd

import (
	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/cli"
)

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configAPIKeyCmd)
	configCmd.AddCommand(configCloudURLCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure this client",
	Long:  "Configure this client",
}

var configAPIKeyCmd = &cobra.Command{
	Use:   "api-key <key>",
	Short: "Set API key",
	Long:  "Set API key",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		config := cli.ReadCLIConfig()
		config.APIKey = args[0]
		cli.WriteCLIConfig(config)
	},
}

var configCloudURLCmd = &cobra.Command{
	Use:   "cloud-url <url>",
	Short: "Set cloud URL",
	Long:  "Set cloud URL",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		config := cli.ReadCLIConfig()
		config.CloudURL = args[0]
		cli.WriteCLIConfig(config)
	},
}
