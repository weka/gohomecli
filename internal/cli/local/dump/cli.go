package dump

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/cli/app/hooks"
	"github.com/weka/gohomecli/internal/local/dump"
	"github.com/weka/gohomecli/internal/utils"
)

var dumpCmd = &cobra.Command{
	Use:       "dump [flags] OUTPUT_ARCHIVE",
	Short:     "Dump cluster information for debugging and save it into OUTPUT_ARCHIVE",
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"OUTPUT_ARCHIVE"},
	RunE:      runDumpScript,
}

var Cli hooks.Cli

var config struct {
	InclideSensitive bool
	FullDiskScan     bool
	Verbose          bool
}

func init() {
	Cli.AddHook(func(appCmd *cobra.Command) {
		appCmd.AddCommand(dumpCmd)

		dumpCmd.Flags().BoolVarP(&config.Verbose, "verbose", "v", false, "Increase verbosity to display debug information during collection phase.")
		dumpCmd.Flags().BoolVar(&config.InclideSensitive, "include-sensitive", false, "Include sensitive data in the archive (e.g., values overrides). Use with caution.")
		dumpCmd.Flags().BoolVar(&config.FullDiskScan, "full-disk-scan", false, "Perform a full disk scan and include the detailed disk usage information in the archive.")
	})
}

func runDumpScript(cmd *cobra.Command, args []string) error {
	opts := []string{"-s", "-", args[0]}

	if config.InclideSensitive {
		opts = append(opts, "--include-sensitive")
	}
	if config.FullDiskScan {
		opts = append(opts, "--full-disk-scan")
	}
	if config.Verbose {
		opts = append(opts, "-v")
	}

	shell := exec.CommandContext(cmd.Context(), "bash", opts...)
	shell.Stdin = bytes.NewReader(dump.DumpScript)

	stdout, err := shell.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := shell.StderrPipe()
	if err != nil {
		return err
	}

	infoLogWriter := utils.NewWriteScanner(func(b []byte) {
		fmt.Println(string(b))
	})
	errorLogWriter := utils.NewWriteScanner(func(b []byte) {
		fmt.Println(string(b))
	})

	go io.Copy(infoLogWriter, stdout)
	go io.Copy(errorLogWriter, stderr)

	return shell.Run()
}
