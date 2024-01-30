package cleanup

import (
	"github.com/spf13/cobra"
	"github.com/weka/gohomecli/internal/local/cleanup"
)

var flags cleanup.ImagesArgs

var imagesCommand = &cobra.Command{
	Use:   "images",
	Short: "clean-up unused images",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cleanup.Images(cmd.Context(), flags)
	},
}

func init() {
	Cli = append(Cli, func(c *cobra.Command) {
		cleanupCmd.AddCommand(imagesCommand)

		imagesCommand.Flags().BoolVar(&flags.Force, "all", false, "remove all non-running images")
		imagesCommand.Flags().BoolVar(&flags.DryRun, "dry-run", false, "only print images that would be removed")
	})
}
