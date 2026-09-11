package embeds

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// frontendFiles is populated by the container build before the Go binary is
// compiled. The tracked placeholder keeps ordinary Go builds reproducible.
//
//go:embed all:frontend
var frontendFiles embed.FS

func FrontendHandler(api http.Handler) http.Handler {
	assets, err := fs.Sub(frontendFiles, "frontend")
	if err != nil {
		panic(err)
	}
	return newFrontendHandler(assets, api)
}

func newFrontendHandler(assets fs.FS, api http.Handler) http.Handler {
	files := http.FileServer(http.FS(assets))
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if isBackendRequest(request.URL.Path) || (request.Method != http.MethodGet && request.Method != http.MethodHead) {
			api.ServeHTTP(writer, request)
			return
		}

		assetPath := strings.TrimPrefix(path.Clean("/"+request.URL.Path), "/")
		if assetPath != "" {
			if info, statErr := fs.Stat(assets, assetPath); statErr == nil && !info.IsDir() {
				writer.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				files.ServeHTTP(writer, request)
				return
			}
			if path.Ext(assetPath) != "" {
				http.NotFound(writer, request)
				return
			}
		}

		if _, statErr := fs.Stat(assets, "index.html"); statErr != nil {
			http.Error(writer, "frontend assets are not embedded", http.StatusServiceUnavailable)
			return
		}
		clone := request.Clone(request.Context())
		clone.URL.Path = "/"
		writer.Header().Set("Cache-Control", "no-cache")
		files.ServeHTTP(writer, clone)
	})
}

func isBackendRequest(requestPath string) bool {
	return requestPath == "/livez" || requestPath == "/readyz" ||
		requestPath == "/api" || strings.HasPrefix(requestPath, "/api/") ||
		requestPath == "/v1" || strings.HasPrefix(requestPath, "/v1/")
}
