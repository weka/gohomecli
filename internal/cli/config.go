package cli

import (
	"fmt"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/weka/gohomecli/internal/env"
	"github.com/weka/gohomecli/internal/utils"
)

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configAPIKeyCmd)
	configCmd.AddCommand(configCloudURLCmd)
	configCmd.AddCommand(configDefaultSiteCmd)

	configCmd.AddCommand(configSiteCmd)
	configSiteCmd.AddCommand(configSiteListCmd)
	configSiteCmd.AddCommand(configSiteAddCmd)
	configSiteCmd.AddCommand(configSiteUpdateCmd)
	configSiteCmd.AddCommand(configSiteRemoveCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration commands",
	Long:  "Configuration commands",
}

var configAPIKeyCmd = &cobra.Command{
	Use:   "api-key <key>",
	Short: "Set API key for default site",
	Long:  "Set API key for default site",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		apiKey := args[0]
		env.UpdateSiteConfig(func(siteConfig *env.SiteConfig) error {
			siteConfig.APIKey = apiKey
			return nil
		})
		utils.UserNote("Updated API key for site \"%s\"", env.SiteName) // do not print the key
	},
}

var configCloudURLCmd = &cobra.Command{
	Use:   "cloud-url <url>",
	Short: "Set cloud URL for default site",
	Long:  "Set cloud URL for default site",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cloudURL := args[0]
		env.UpdateSiteConfig(func(siteConfig *env.SiteConfig) error {
			siteConfig.CloudURL = cloudURL
			return nil
		})
		utils.UserNote("Updated cloud URL for site \"%s\": %s", env.SiteName, cloudURL)
	},
}

var configDefaultSiteCmd = &cobra.Command{
	Use:   "default-site <site>",
	Short: "Set default site",
	Long:  "Set default site",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		siteName := args[0]
		env.UpdateConfig(func(config *env.Config) error {
			_, exists := config.Sites[siteName]
			if !exists {
				return fmt.Errorf("no such site: \"%s\"", siteName)
			}
			config.DefaultSite = siteName
			return nil
		})
		utils.UserNote("Set default site configuration: \"%s\"", siteName)
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
		env.UpdateConfig(func(config *env.Config) error {
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

var configSiteUpdateCmd = &cobra.Command{
	Use:   "update <site> <cloud-url> <api-key>",
	Short: "Update site configuration",
	Long:  "Update site configuration",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		siteName, cloudURL, apiKey := args[0], args[1], args[2]
		env.UpdateConfig(func(config *env.Config) error {
			site, exists := config.Sites[siteName]
			if !exists {
				return fmt.Errorf("no such site: \"%s\"", siteName)
			}
			site.APIKey = apiKey
			site.CloudURL = cloudURL
			return nil
		})
		utils.UserNote("Updated site configuration: \"%s\"", siteName)
	},
}

var configSiteRemoveCmd = &cobra.Command{
	Use:   "remove <site>",
	Short: "Remove a configured site",
	Long:  "Remove a configured site",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		siteName := args[0]
		env.UpdateConfig(func(config *env.Config) error {
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
