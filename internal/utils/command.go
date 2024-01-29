package utils

import (
	"bufio"
	"context"
	"io"
	"os/exec"
	"sync"
)

type commandOpt func(*WrappedCmd) error

// WithStdoutLogger returns a command option which sends stdout to channel in callback
var WithStdoutReader = func(cb func(lines chan []byte)) func(cmd *WrappedCmd) error {
	return func(cmd *WrappedCmd) error {
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return err
		}

		return cmd.startReader(stdout, cb)
	}
}

// WithStderrLogger returns a command option which sends stderr to channel in callback
var WithStderrReader = func(cb func(lines chan []byte)) func(cmd *WrappedCmd) error {
	return func(cmd *WrappedCmd) error {
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return err
		}

		return cmd.startReader(stderr, cb)
	}
}

var WithStdin = func(stdin io.Reader) func(cmd *WrappedCmd) error {
	return func(cmd *WrappedCmd) error {
		cmd.Stdin = stdin
		return nil
	}
}

type WrappedCmd struct {
	*exec.Cmd
	wg sync.WaitGroup
}

// startReader starts a goroutine which reads from reader and sends lines to callback
func (c *WrappedCmd) startReader(reader io.Reader, cb func(chan []byte)) error {
	// we need to wait for reader to be closed before we can wait for the command to finish
	c.wg.Add(1)

	lines := make(chan []byte)

	// run callback
	go func() {
		cb(lines)
		c.wg.Done()
	}()

	// run scanner
	go func() {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			bytes := scanner.Bytes()
			if len(bytes) > 0 {
				lines <- bytes
			}
		}
		close(lines)
	}()

	return nil
}

func (c *WrappedCmd) Wait() error {
	c.wg.Wait()
	return c.Cmd.Wait()
}

func ExecCommand(ctx context.Context, name string, args []string, opts ...commandOpt) (*WrappedCmd, error) {
	cmd := WrappedCmd{
		Cmd: exec.CommandContext(ctx, name, args...),
	}

	for _, opt := range opts {
		if err := opt(&cmd); err != nil {
			return nil, err
		}
	}
	return &cmd, cmd.Start()
}

type WriterFunc struct {
	*sync.WaitGroup

	io.Writer
	io.Closer
	ErrCloser interface {
		CloseWithError(err error) error
	}
}

// NewWritterFunc returns a helper which runs callback for each line, written to WriterFunc
func NewWritterFunc(readers ...func([]byte)) WriterFunc {
	var wg sync.WaitGroup

	wg.Add(1)
	reader, writer := io.Pipe()
	go func() {
		scan := bufio.NewScanner(reader)
		for scan.Scan() {
			for _, cb := range readers {
				cb(scan.Bytes())
			}
		}
		wg.Done()
	}()

	return WriterFunc{
		WaitGroup: &wg,
		Writer:    writer,
		Closer:    writer,
		ErrCloser: writer,
	}
}
