package cmd

import (
	"errors"
	"github.com/spf13/cobra"

	"home.weka.io/cli"
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
	Use:   "api-key",
	Short: "Set API key",
	Long:  "Set API key",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("requires exactly 1 argument (API key)")
		}
		return nil
		// return fmt.Errorf("invalid color specified: %s", args[0])
	},
	Run: func(cmd *cobra.Command, args []string) {
		config := cli.ReadCLIConfig()
		config.APIKey = args[0]
		cli.WriteCLIConfig(config)
	},
}

var configCloudURLCmd = &cobra.Command{
	Use:   "cloud-url",
	Short: "Set cloud URL",
	Long:  "Set cloud URL",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("requires exactly 1 argument (API key)")
		}
		return nil
		// return fmt.Errorf("invalid color specified: %s", args[0])
	},
	Run: func(cmd *cobra.Command, args []string) {
		config := cli.ReadCLIConfig()
		config.CloudURL = args[0]
		cli.WriteCLIConfig(config)
	},
}
