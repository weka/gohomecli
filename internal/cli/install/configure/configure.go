package configure

import (
	"errors"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/cli/app/hooks"
	"github.com/weka/gohomecli/internal/install/web"
	"github.com/weka/gohomecli/internal/utils"
)

var (
	Cli    hooks.Cli
	logger = utils.GetLogger("Configurer")
)

var (
	configureCmd = &cobra.Command{
		Use:   "configure",
		Short: "Configure Weka Home interactively",
	}
	webArgs = struct{ bindAddr string }{}
	webCmd  = &cobra.Command{
		Use:   "web",
		Short: "Configure Weka Home using web interface",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.Info().Str("bindAddr", webArgs.bindAddr).Msg("Starting web server")
			if err := web.ServeConfigurer(cmd.Context(), webArgs.bindAddr); errors.Is(err, http.ErrServerClosed) {
				return nil
			} else {
				return err
			}
		},
	}
)

func init() {
	Cli.AddHook(func(appCmd *cobra.Command) {
		appCmd.AddCommand(configureCmd)

		if web.IsEnabled() {
			configureCmd.AddCommand(webCmd)
			webCmd.Flags().StringVarP(&webArgs.bindAddr, "bind-addr", "b", ":8080", "Bind address for web server including port")
		}
	})
}
