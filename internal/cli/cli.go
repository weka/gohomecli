package cli

import (
	"errors"

	"github.com/weka/gohomecli/internal/cli/api"
	"github.com/weka/gohomecli/internal/cli/app"
	"github.com/weka/gohomecli/internal/cli/config"
	"github.com/weka/gohomecli/internal/cli/install"
	"github.com/weka/gohomecli/internal/utils"
)

func init() {
	api.Cli.InitCobra(app.Cmd())
	config.Cli.InitCobra(app.Cmd())
	install.Cli.InitCobra(app.Cmd())
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := app.Cmd().Execute(); err != nil {
		if errors.Is(err, utils.ErrValidationFailed) {
			app.Cmd().Usage()
		}

		utils.UserError(err.Error())
	}
}
