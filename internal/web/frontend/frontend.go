package frontend

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
)

//go:embed build
var webBuild embed.FS

var ServeFrontend http.Handler

func init() {
	serverRoot, err := fs.Sub(webBuild, "build")
	if err != nil {
		log.Fatal(err)
	}

	ServeFrontend = http.FileServer(http.FS(serverRoot))
}
