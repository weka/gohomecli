package utils

import (
	"bufio"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Log levels to be used by main, or other entry points in this project
const (
	DebugLevel = zerolog.DebugLevel
	InfoLevel  = zerolog.InfoLevel
	WarnLevel  = zerolog.WarnLevel
)

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	output := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}
	log.Logger = zerolog.New(output).With().Timestamp().Logger()
	SetGlobalLoggingLevel(WarnLevel)
}

// SetGlobalLoggingLevel should be invoked once by each entry point
func SetGlobalLoggingLevel(level zerolog.Level) {
	zerolog.SetGlobalLevel(level)
}

// GetLogger returns a new logger instance with the specified component
func GetLogger(component string) zerolog.Logger {
	return log.With().Str("component", component).Logger()
}

type WriteScanner struct {
	io.Writer
	io.Closer
	ErrCloser interface {
		CloseWithError(err error) error
	}
}

func NewWriteScanner(readers ...func([]byte)) WriteScanner {
	reader, writer := io.Pipe()
	go func() {
		scan := bufio.NewScanner(reader)
		for scan.Scan() {
			for _, cb := range readers {
				cb(scan.Bytes())
			}
		}
	}()

	return WriteScanner{
		Writer:    writer,
		Closer:    writer,
		ErrCloser: writer,
	}
}
