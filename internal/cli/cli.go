package cli

import (
	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/cli/app"

	_ "github.com/weka/gohomecli/internal/cli/api"
	_ "github.com/weka/gohomecli/internal/cli/config"
	_ "github.com/weka/gohomecli/internal/cli/k3s"
)

func init() {
	app.AppCmd.AddGroup(&cobra.Group{
		ID:    "API",
		Title: "WekaHome API commands",
	})
}

func Execute() {
	app.Execute()
}
