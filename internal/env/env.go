package env

import "os"

type VersionInfoAttributes struct {
	Name      string
	BuildTime string
}

var VersionInfo VersionInfoAttributes

var IsInteractiveTerminal bool

func init() {
	fileInfo, _ := os.Stdout.Stat()
	IsInteractiveTerminal = (fileInfo.Mode() & os.ModeCharDevice) != 0
	//IsColorOutputSupported = IsInteractiveTerminal
}
