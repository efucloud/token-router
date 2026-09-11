package v1

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/efucloud/token-router/pkg/services"
	restful "github.com/emicklei/go-restful/v3"
)

func TestWriteQuotaErrorIncludesRateLimitHeaders(t *testing.T) {
	recorder := httptest.NewRecorder()
	response := restful.NewResponse(recorder)
	response.SetRequestAccepts(restful.MIME_JSON)
	resetAt := time.Now().Add(30 * time.Second)

	writeQuotaError(response, services.GatewayQuotaError{Dimension: "API key RPM", Limit: 60, ResetAt: resetAt})

	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", recorder.Code)
	}
	if recorder.Header().Get("X-RateLimit-Limit") != "60" {
		t.Fatalf("unexpected rate limit header: %q", recorder.Header().Get("X-RateLimit-Limit"))
	}
	if recorder.Header().Get("Retry-After") == "" || recorder.Header().Get("X-RateLimit-Reset") == "" {
		t.Fatalf("missing retry metadata: %#v", recorder.Header())
	}
}

func TestGatewayRoutesAcceptStreamingMediaTypes(t *testing.T) {
	tests := []struct {
		path   string
		accept string
	}{
		{path: "/v1/chat/completions", accept: "application/x-ndjson"},
		{path: "/v1/responses", accept: "text/event-stream"},
	}
	for _, test := range tests {
		t.Run(test.accept, func(t *testing.T) {
			container := restful.NewContainer()
			ws := new(restful.WebService)
			ws.Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)
			GatewayResource{Svc: services.GatewayService{}}.AddWebService(ws)
			container.Add(ws)

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, test.path, nil)
			request.Header.Set("Accept", test.accept)
			request.Header.Set("Content-Type", restful.MIME_JSON)
			container.ServeHTTP(recorder, request)

			if recorder.Code == http.StatusNotAcceptable {
				t.Fatalf("streaming media type %q was rejected", test.accept)
			}
			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("expected request to reach authentication, got %d: %s", recorder.Code, recorder.Body.String())
			}
		})
	}
}
