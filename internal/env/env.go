package env

import (
	"os"

	"github.com/weka/gohomecli/internal/utils"
)

type VersionInfoAttributes struct {
	Name      string
	BuildTime string
}

var VersionInfo VersionInfoAttributes

var IsInteractiveTerminal bool

var (
	ColorMode      string
	VerboseLogging bool
)

var validColors = map[string]bool{
	"auto":   true,
	"always": true,
	"never":  true,
}

func init() {
	fileInfo, _ := os.Stdout.Stat()
	IsInteractiveTerminal = (fileInfo.Mode() & os.ModeCharDevice) != 0
	// IsColorOutputSupported = IsInteractiveTerminal
}

func IsValidColorMode() bool {
	return validColors[ColorMode]
}

func InitEnv() {
	switch ColorMode {
	case "always":
		utils.IsColorOutputSupported = true
	case "never":
		utils.IsColorOutputSupported = false
	case "auto":
		utils.IsColorOutputSupported = IsInteractiveTerminal
	}
	initLogging()
}

func initLogging() {
	if VerboseLogging {
		utils.SetLoggingLevel(utils.DebugLevel)
	}
}
