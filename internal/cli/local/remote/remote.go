// Package remote provides CLI commands for managing remote tmate sessions
package remote

import (
	"errors"
	"fmt"

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

	// validOutputFormats defines the allowed output format values
	validOutputFormats = []string{"table", "json", "yaml"} //nolint:gochecknoglobals // used by multiple commands

	// ErrInvalidOutputFormat is returned when an invalid output format is specified
	ErrInvalidOutputFormat = errors.New("invalid output format")
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

// validateOutputFormat checks if the output format is valid
func validateOutputFormat(format string) error {
	for _, valid := range validOutputFormats {
		if format == valid {
			return nil
		}
	}

	return fmt.Errorf("%w: %q (valid: table, json, yaml)", ErrInvalidOutputFormat, format)
}
