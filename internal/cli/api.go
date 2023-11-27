package cli

import (
	"github.com/spf13/cobra"
	"github.com/weka/gohomecli/internal/cli/app"
)

func init() {
	app.AppCmd.AddGroup(&cobra.Group{
		ID:    "API",
		Title: "WekaHome API commands",
	})
}
