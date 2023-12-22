package cleanup

import (
	"github.com/spf13/cobra"
	"github.com/weka/gohomecli/internal/local/cleanup"
)

var config struct {
	LocalPath string
}

var localStorageCmd = &cobra.Command{
	Use:   "local-storage",
	Short: "cleans up unused volumes from local path provisioner",
	RunE: func(cmd *cobra.Command, args []string) error {
		if config.LocalPath != "" {
			cleanup.SetLocalStoragePath(config.LocalPath)
		}

		return cleanup.LocalStorage(cmd.Context())
	},
}

func init() {
	Cli = append(Cli, func(c *cobra.Command) {
		cleanupCmd.AddCommand(localStorageCmd)

		cleanupCmd.Flags().StringVar(&config.LocalPath, "path", "", "directory to local-path-provisioner")
		cleanupCmd.Flags().MarkHidden("bundle")
	})
}
