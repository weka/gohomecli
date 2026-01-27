// Package remote provides CLI commands for managing remote tmate sessions
package remote

import (
	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/cli/app/hooks"
	"github.com/weka/gohomecli/internal/utils"
)

var (
	logger = utils.GetLogger("Remote") //nolint:gochecknoglobals // standard pattern

	// RemoteAccessGroup is the command group for remote access commands
	RemoteAccessGroup = cobra.Group{ //nolint:gochecknoglobals // cobra pattern
		ID:    "remote-access",
		Title: "Remote Access Commands",
	}

	// Cli is the hooks.Cli instance for remote access commands
	Cli hooks.Cli //nolint:gochecknoglobals // cobra pattern
)

//nolint:gochecknoinits // cobra CLI pattern
func init() {
	Cli.AddHook(func(appCmd *cobra.Command) {
		appCmd.AddGroup(&RemoteAccessGroup)

		remoteCmd := &cobra.Command{
			Use:     "remote-access",
			Short:   "Manage remote tmate sessions",
			Long:    "Start, stop, and manage remote tmate sessions and recordings for cluster debugging",
			GroupID: RemoteAccessGroup.ID,
		}

		remoteCmd.AddCommand(newStartCmd())
		remoteCmd.AddCommand(newStopCmd())
		remoteCmd.AddCommand(newListCmd())
		remoteCmd.AddCommand(newListRecordingsCmd())
		remoteCmd.AddCommand(newCopyRecordingCmd())

		appCmd.AddCommand(remoteCmd)
	})
}
