//go:build web

package frontend

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
)

//go:embed build
var webBuild embed.FS

func Router() http.Handler {
	serverRoot, err := fs.Sub(webBuild, "build")
	if err != nil {
		log.Fatal(err)
	}

	return http.FileServer(http.FS(serverRoot))
}
