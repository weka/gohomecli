package config

import (
	"fmt"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/env"
	"github.com/weka/gohomecli/internal/utils"
)

var inits []func()

func init() {
	inits = append(inits, func() {
		appCmd.AddGroup(ConfigGroup)
		appCmd.AddCommand(configCmd)

		configCmd.AddCommand(configSiteCmd)
		configCmd.AddCommand(configUpdateCmd)
		configUpdateCmd.Flags().StringVar(&configUpdateCmdArgs.cloudURL, "cloud-url", "",
			"set cloud URL")
		configUpdateCmd.Flags().StringVar(&configUpdateCmdArgs.apiKey, "api-key", "",
			"set API key")
		configCmd.AddCommand(configDefaultSiteCmd)
		configSiteCmd.AddCommand(configSiteListCmd)
		configSiteCmd.AddCommand(configSiteAddCmd)
		configSiteCmd.AddCommand(configSiteRemoveCmd)
	})
}

var appCmd *cobra.Command

var ConfigGroup = &cobra.Group{ID: "Config", Title: "CLI Configuration"}

var configCmd = &cobra.Command{
	Use:     "config",
	Short:   "Configuration commands",
	Long:    "Configuration commands",
	GroupID: "Config",
}

var configUpdateCmdArgs = struct {
	cloudURL string
	apiKey   string
}{}

var configUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update configuration",
	Long:  "Update configuration",
	Run: func(cmd *cobra.Command, args []string) {
		env.UpdateConfig(func(config *env.Config, siteConfig *env.SiteConfig) error {
			atLeastOne := false
			if configUpdateCmdArgs.apiKey != "" {
				siteConfig.APIKey = configUpdateCmdArgs.apiKey
				atLeastOne = true
			}
			if configUpdateCmdArgs.cloudURL != "" {
				siteConfig.CloudURL = configUpdateCmdArgs.cloudURL
				atLeastOne = true
			}
			if !atLeastOne {
				utils.UserError("at least one configuration value must be set, see help for more info")
			}
			return nil
		})
		utils.UserNote("Updated site configuration: \"%s\"", env.SiteName)
	},
}

var configDefaultSiteCmd = &cobra.Command{
	Use:   "default-site <site>",
	Short: "Set default site",
	Long:  "Set default site",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		siteName := args[0]
		env.UpdateConfig(func(config *env.Config, siteConfig *env.SiteConfig) error {
			_, exists := config.Sites[siteName]
			if !exists {
				return fmt.Errorf("no such site: \"%s\"", siteName)
			}
			config.DefaultSite = siteName
			return nil
		})
		utils.UserNote("Default site configuration set: \"%s\"", siteName)
	},
}

var configSiteCmd = &cobra.Command{
	Use:   "site",
	Short: "Site configuration commands",
	Long:  "Site configuration commands",
}

var configSiteListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured sites",
	Long:  "List configured sites",
	Run: func(cmd *cobra.Command, args []string) {
		utils.RenderTable([]string{"Name", "URL", "Default"}, func(table *tablewriter.Table) {
			for name, site := range env.CurrentConfig.Sites {
				var defaultSymbol string
				if name == env.CurrentConfig.DefaultSite {
					defaultSymbol = "*"
				} else {
					defaultSymbol = ""
				}
				table.Append([]string{name, site.CloudURL, defaultSymbol})
			}
		})
	},
}

var configSiteAddCmd = &cobra.Command{
	Use:   "add <site> <cloud-url> <api-key>",
	Short: "Configure a new site",
	Long:  "Configure a new site",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		siteName, cloudURL, apiKey := args[0], args[1], args[2]
		env.UpdateConfig(func(config *env.Config, siteConfig *env.SiteConfig) error {
			_, exists := config.Sites[siteName]
			if exists {
				return fmt.Errorf("site already exists: \"%s\"", siteName)
			}
			config.Sites[siteName] = &env.SiteConfig{APIKey: apiKey, CloudURL: cloudURL}
			return nil
		})
		utils.UserNote("Added site configuration: \"%s\"", siteName)
	},
}

var configSiteRemoveCmd = &cobra.Command{
	Use:   "remove <site>",
	Short: "Remove a configured site",
	Long:  "Remove a configured site",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		siteName := args[0]
		env.UpdateConfig(func(config *env.Config, siteConfig *env.SiteConfig) error {
			_, exists := config.Sites[siteName]
			if !exists {
				return fmt.Errorf("no such site: \"%s\"", siteName)
			}
			delete(config.Sites, siteName)
			return nil
		})
		utils.UserNote("Removed site configuration: \"%s\"", siteName)
	},
}

func Init(cmd *cobra.Command) {
	appCmd = cmd
	for i := range inits {
		inits[i]()
	}
}
