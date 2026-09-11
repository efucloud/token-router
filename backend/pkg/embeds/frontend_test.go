package embeds

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestFrontendHandlerServesAssetsAndSPARoutes(t *testing.T) {
	assets := fstest.MapFS{
		"index.html":        &fstest.MapFile{Data: []byte("<html>console</html>")},
		"umi.abc123.js":     &fstest.MapFile{Data: []byte("console.log('ok')")},
		"nested/readme.txt": &fstest.MapFile{Data: []byte("asset")},
	}
	api := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusTeapot)
		_, _ = writer.Write([]byte("api"))
	})
	handler := newFrontendHandler(fs.FS(assets), api)

	for _, requestPath := range []string{"/", "/personal/chat", "/oauth/callback"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, requestPath, nil))
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "console") {
			t.Fatalf("SPA route %q was not served: status=%d body=%q", requestPath, response.Code, response.Body.String())
		}
		if response.Header().Get("Cache-Control") != "no-cache" {
			t.Fatalf("SPA route %q must not be cached: %#v", requestPath, response.Header())
		}
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/umi.abc123.js", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "console.log") {
		t.Fatalf("static asset was not served: status=%d body=%q", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Header().Get("Cache-Control"), "immutable") {
		t.Fatalf("static asset cache policy is missing: %#v", response.Header())
	}

	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/missing.js", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("missing static asset must return 404, got %d", response.Code)
	}
}

func TestFrontendHandlerDelegatesBackendRequests(t *testing.T) {
	assets := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("frontend")}}
	api := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusTeapot)
	})
	handler := newFrontendHandler(fs.FS(assets), api)

	for _, requestPath := range []string{"/api/v1/health", "/v1/models", "/livez", "/readyz"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, requestPath, nil))
		if response.Code != http.StatusTeapot {
			t.Fatalf("backend route %q was not delegated: %d", requestPath, response.Code)
		}
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/personal/chat", nil))
	if response.Code != http.StatusTeapot {
		t.Fatalf("non-GET frontend request was not delegated: %d", response.Code)
	}
}
