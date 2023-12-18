package dump

import (
	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/cli/app/hooks"
	"github.com/weka/gohomecli/internal/local/dump"
)

var dumpCmd = &cobra.Command{
	Use:       "dump [flags] OUTPUT_ARCHIVE",
	Short:     "Dump cluster information for debugging and save it into OUTPUT_ARCHIVE",
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"OUTPUT_ARCHIVE"},
	RunE: func(cmd *cobra.Command, args []string) error {
		config.Output = args[0]
		return dump.Dump(cmd.Context(), config)
	},
}

var (
	Cli    hooks.Cli
	config dump.Config
)

func init() {
	Cli.AddHook(func(appCmd *cobra.Command) {
		appCmd.AddCommand(dumpCmd)

		dumpCmd.Flags().BoolVarP(&config.Verbose, "verbose", "v", false, "Increase verbosity to display debug information during collection phase.")
		dumpCmd.Flags().BoolVar(&config.InclideSensitive, "include-sensitive", false, "Include sensitive data in the archive (e.g., values overrides). Use with caution.")
		dumpCmd.Flags().BoolVar(&config.FullDiskScan, "full-disk-scan", false, "Perform a full disk scan and include the detailed disk usage information in the archive.")
	})
}
