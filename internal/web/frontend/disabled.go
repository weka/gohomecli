//go:build !web

package frontend

import "net/http"

func Router() http.Handler {
	return nil
}
