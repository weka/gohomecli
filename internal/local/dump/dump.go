package dump

import (
	"bytes"
	"context"
	_ "embed"
	"io"
	"os/exec"

	"github.com/weka/gohomecli/internal/utils"
)

//go:embed dump.sh
var script []byte

var logger = utils.GetLogger("dump")

type Config struct {
	Output           string
	IncludeSensitive bool
	FullDiskScan     bool
	Verbose          bool
}

func (c Config) toArgs() []string {
	opts := []string{c.Output}

	if c.IncludeSensitive {
		opts = append(opts, "--include-sensitive")
	}
	if c.FullDiskScan {
		opts = append(opts, "--full-disk-scan")
	}
	if c.Verbose {
		opts = append(opts, "-v")
	}
	return opts
}

func Dump(ctx context.Context, config Config) error {
	bashArgs := []string{"-s", "-"}

	shell := exec.CommandContext(ctx, "bash", append(bashArgs, config.toArgs()...)...)
	shell.Stdin = bytes.NewReader(script)

	stdout, err := shell.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := shell.StderrPipe()
	if err != nil {
		return err
	}

	infoLogWriter := utils.NewWritterFunc(func(b []byte) {
		logger.WithLevel(utils.InfoLevel).Msg(string(b))
	})
	errorLogWriter := utils.NewWritterFunc(func(b []byte) {
		logger.WithLevel(utils.WarnLevel).Msg(string(b))
	})

	go io.Copy(infoLogWriter, stdout)
	go io.Copy(errorLogWriter, stderr)

	return shell.Run()
}
