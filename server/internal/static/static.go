package static

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dist
var distFS embed.FS

func Handler() (http.Handler, error) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, err
	}

	return newHandler(sub), nil
}

func newHandler(sub fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "index.html" {
			path = ""
		}

		if path != "" {
			if _, err := sub.Open(path); err != nil {
				r.URL.Path = "/200.html"
				fileServer.ServeHTTP(w, r)
				return
			}

			r.URL.Path = "/" + path
		} else {
			r.URL.Path = "/"
		}

		fileServer.ServeHTTP(w, r)
	})
}

func MustHandler() http.Handler {
	h, err := Handler()
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "static assets not embedded", http.StatusNotFound)
		})
	}
	return h
}
