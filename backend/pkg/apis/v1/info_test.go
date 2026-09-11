package v1

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/efucloud/token-router/pkg/config"
	restful "github.com/emicklei/go-restful/v3"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func readinessResponse(t *testing.T) *httptest.ResponseRecorder {
	t.Helper()
	request := restful.NewRequest(httptest.NewRequest(http.MethodGet, "/readyz", nil))
	recorder := httptest.NewRecorder()
	response := restful.NewResponse(recorder)
	response.SetRequestAccepts(restful.MIME_JSON)
	(InfoResource{}).readiness(request, response)
	return recorder
}

func TestReadinessChecksDatabaseSchemaAndGatewaySecret(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open readiness database: %v", err)
	}
	for _, table := range []string{"account", "ai_model", "channel", "model_route", "api_token", "usage_log", "audit_log"} {
		if err = db.Exec("CREATE TABLE " + table + " (id TEXT PRIMARY KEY)").Error; err != nil {
			t.Fatalf("create %s: %v", table, err)
		}
	}
	previousDB := config.DBConnect
	previousConfig := config.ApplicationConfig
	config.DBConnect = db
	config.ApplicationConfig = &config.Config{Gateway: config.GatewayConfig{
		SecretKey: base64.StdEncoding.EncodeToString([]byte("12345678901234567890123456789012")),
	}}
	t.Cleanup(func() {
		config.DBConnect = previousDB
		config.ApplicationConfig = previousConfig
	})

	if response := readinessResponse(t); response.Code != http.StatusOK {
		t.Fatalf("expected ready response, got %d: %s", response.Code, response.Body.String())
	}
	config.ApplicationConfig.Gateway.SecretKey = "invalid"
	if response := readinessResponse(t); response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected invalid secret to fail readiness, got %d: %s", response.Code, response.Body.String())
	}
}
