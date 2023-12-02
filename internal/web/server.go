package web

import (
	"net/http"

	"github.com/weka/gohomecli/internal/web/frontend"
)

func ListenAndServe(addr string) error {
	router := http.NewServeMux()
	router.Handle("/", frontend.ServeFrontend)
	router.HandleFunc("/api/calculate", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	})

	return http.ListenAndServe(addr, router)
}
