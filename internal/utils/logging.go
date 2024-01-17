package utils

import (
	"bufio"
	"io"
	"os"
	"os/exec"
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

const debugLog = "/var/log/homecli.log"

var stdoutWriter *zerolog.FilteredLevelWriter

func init() {
	var logWriter zerolog.LevelWriter

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	stdoutWriter = &zerolog.FilteredLevelWriter{
		Writer: zerolog.LevelWriterAdapter{
			Writer: zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339},
		},
		Level: WarnLevel,
	}

	logWriter = stdoutWriter

	f, err := os.OpenFile(debugLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0660)
	if err == nil {
		debugWriter := zerolog.FilteredLevelWriter{
			Writer: zerolog.LevelWriterAdapter{Writer: f},
			Level:  DebugLevel,
		}
		logWriter = zerolog.MultiLevelWriter(logWriter, &debugWriter)
	}

	log.Logger = zerolog.New(logWriter).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(DebugLevel)

	SetLoggingLevel(WarnLevel)
}

// SetLoggingLevel should be invoked once by each entry point
func SetLoggingLevel(level zerolog.Level) {
	stdoutWriter.Level = level
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

func LineLogger(logger zerolog.Logger, level zerolog.Level, cb ...func(*zerolog.Event)) func([]byte) {
	return func(b []byte) {
		event := logger.WithLevel(level)
		for _, c := range cb {
			c(event)
		}
		event.Msg(string(b))
	}
}

var WithStdoutLogger = func(logger zerolog.Logger, level zerolog.Level, cb ...func(*zerolog.Event)) func(*exec.Cmd) error {
	return WithStdoutReader(LineLogger(logger, level, cb...))
}

var WithStderrLogger = func(logger zerolog.Logger, level zerolog.Level, cb ...func(*zerolog.Event)) func(*exec.Cmd) error {
	return WithStderrReader(LineLogger(logger, level, cb...))
}
