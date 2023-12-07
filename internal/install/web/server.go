//go:build web

package web

import (
	"context"
	"net/http"

	"github.com/weka/gohomecli/internal/utils"

	"github.com/weka/gohomecli/internal/install/web/api"

	"github.com/weka/gohomecli/internal/install/web/frontend"
)

var logger = utils.GetLogger("Configurer")

func ServeConfigurer(ctx context.Context, addr string) error {
	router := http.NewServeMux()
	router.Handle("/", frontend.Router())
	router.Handle("/api/", api.Router())

	server := http.Server{Addr: addr, Handler: router}

	go func() {
		<-ctx.Done()
		err := server.Shutdown(ctx)
		if err != nil {
			logger.Fatal().Err(err).Msg("Failed to shutdown web server")
		}
	}()

	return server.ListenAndServe()
}

func IsEnabled() bool {
	return true
}
