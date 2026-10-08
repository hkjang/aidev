package webui

import (
	"bytes"
	"embed"
	"io/fs"
	"net/http"
	"strings"
	"time"
)

// console contains the production React build.
//
//go:embed dist
var console embed.FS

// Decorator rewrites the console shell before it is served. It receives the
// embedded index.html and returns what the browser gets; returning the same
// bytes means the shell is served exactly as built.
type Decorator func(request *http.Request, page []byte) []byte

func Handler() http.Handler {
	return Decorated(nil)
}

// Decorated serves the console with the shell passed through decorate, which
// is how the visitor tracking snippet reaches the page without the React build
// knowing about it.
func Decorated(decorate Decorator) http.Handler {
	root, err := fs.Sub(console, "dist")
	if err != nil {
		panic(err)
	}
	shell, err := fs.ReadFile(root, "index.html")
	if err != nil {
		panic(err)
	}
	files := http.FileServer(http.FS(root))
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if strings.HasPrefix(request.URL.Path, "/api/") ||
			strings.HasPrefix(request.URL.Path, "/v1/") ||
			strings.HasPrefix(request.URL.Path, "/health/") {
			response.Header().Set("Content-Type", "application/json")
			response.WriteHeader(http.StatusNotFound)
			_, _ = response.Write([]byte(`{"error":{"code":"NOT_FOUND","message":"The requested API resource does not exist."}}`))
			return
		}
		path := strings.TrimPrefix(request.URL.Path, "/")
		// The shell itself is never served as a plain file: a direct request
		// for /index.html must carry the same snippet as /.
		if path != "" && path != "index.html" {
			if _, err := fs.Stat(root, path); err == nil {
				files.ServeHTTP(response, request)
				return
			}
		}
		response.Header().Set("Cache-Control", "no-cache")
		if decorate != nil {
			if page := decorate(request, shell); !bytes.Equal(page, shell) {
				response.Header().Set("Content-Type", "text/html; charset=utf-8")
				http.ServeContent(
					response, request, "index.html", time.Time{},
					bytes.NewReader(page),
				)
				return
			}
		}
		request.URL.Path = "/"
		files.ServeHTTP(response, request)
	})
}
