// Package debug provides a CLI command to collect debug information from the local cluster.
package debug

import (
	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/cli/app/hooks"
	"github.com/weka/gohomecli/internal/local/dump"
)

// Cli is debug package CLI instance
// nolint:gochecknoglobals // this shouldnt be a global variable, but this is a much deeper refactor
var Cli hooks.Cli

// nolint:gochecknoinits // bad practice, but this is a much deeper refactor
func init() {
	var config dump.Config
	dumpCmd := &cobra.Command{
		Use:       "collect-debug-info [flags] OUTPUT_ARCHIVE",
		Short:     "Dump cluster information for debugging and save it into OUTPUT_ARCHIVE",
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"OUTPUT_ARCHIVE"},
		RunE: func(cmd *cobra.Command, args []string) error {
			config.Output = args[0]

			return dump.Dump(cmd.Context(), config)
		},
	}

	Cli.AddHook(func(appCmd *cobra.Command) {
		appCmd.AddCommand(dumpCmd)

		dumpCmd.Flags().
			BoolVarP(&config.Verbose, "verbose", "v", false, "Increase verbosity to display debug information during collection phase.")
		dumpCmd.Flags().
			BoolVar(&config.IncludeSensitive, "include-sensitive", false, "Include sensitive data in the archive (e.g., values overrides). Use with caution.")
		dumpCmd.Flags().
			BoolVar(&config.FullDiskScan, "full-disk-scan", false, "Perform a full disk scan and include the detailed disk usage information in the archive.")
	})
}
