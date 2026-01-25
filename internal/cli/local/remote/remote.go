// Package remote provides CLI commands for managing remote tmate sessions
package remote

import (
	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/cli/app/hooks"
	"github.com/weka/gohomecli/internal/utils"
)

var logger = utils.GetLogger("Remote")

// CliHook returns a Cobra CLI hook for the remote command
func CliHook() hooks.Cli {
	var cli hooks.Cli

	remoteCmd := &cobra.Command{
		Use:   "remote",
		Short: "Manage remote tmate sessions",
		Long:  "Start, stop, and manage remote tmate sessions for cluster debugging",
	}

	remoteCmd.AddCommand(newStartCmd())
	remoteCmd.AddCommand(newStopCmd())
	remoteCmd.AddCommand(newListCmd())
	remoteCmd.AddCommand(newListRecordingsCmd())
	remoteCmd.AddCommand(newCopyRecordingCmd())

	cli.AddHook(func(localCmd *cobra.Command) {
		localCmd.AddCommand(remoteCmd)
	})

	return cli
}
