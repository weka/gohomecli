package api

import (
	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/cli/app/hooks"
)

func init() {
	Cli.AddHook(func(appCmd *cobra.Command) {
		appCmd.AddGroup(&APIGroup)
	})
}

var APIGroup = cobra.Group{
	ID:    "API",
	Title: "WekaHome API commands",
}

var Cli hooks.Cli
