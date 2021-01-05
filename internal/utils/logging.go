package utils

import (
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
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
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
