package admin

import (
	"net/http"
	"testing"

	"github.com/efucloud/token-router/pkg/config"
	"github.com/emicklei/go-restful/v3"
)

func TestChannelTestRouteUsesGET(t *testing.T) {
	ws := new(restful.WebService)
	ControlPlaneResource{}.AddWebService(ws)
	targetPath := config.APIPrefix + "/channels/{id}/test"
	for _, route := range ws.Routes() {
		if route.Path == targetPath {
			if route.Method != http.MethodGet {
				t.Fatalf("expected %s to use GET, got %s", targetPath, route.Method)
			}
			return
		}
	}
	t.Fatalf("route %s was not registered", targetPath)
}

func TestChannelModelSyncRouteUsesPOST(t *testing.T) {
	ws := new(restful.WebService)
	ControlPlaneResource{}.AddWebService(ws)
	targetPath := config.APIPrefix + "/channels/{id}/models/sync"
	for _, route := range ws.Routes() {
		if route.Path == targetPath {
			if route.Method != http.MethodPost {
				t.Fatalf("expected %s to use POST, got %s", targetPath, route.Method)
			}
			return
		}
	}
	t.Fatalf("route %s was not registered", targetPath)
}
