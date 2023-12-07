//go:build web

package web

import (
	"net/http"

	"github.com/weka/gohomecli/internal/web/api"

	"github.com/weka/gohomecli/internal/web/frontend"
)

func ListenAndServe(addr string) error {
	router := http.NewServeMux()
	router.Handle("/", frontend.Router())
	router.Handle("/api/", api.Router())

	return http.ListenAndServe(addr, router)
}

func IsEnabled() bool {
	return true
}
