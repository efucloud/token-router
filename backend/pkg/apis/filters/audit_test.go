package filters

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/efucloud/token-router/pkg/config"
	"github.com/efucloud/token-router/pkg/models/daos"
	restful "github.com/emicklei/go-restful/v3"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestAuditRecordsMutationWithoutSensitiveBody(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open audit database: %v", err)
	}
	if err = db.AutoMigrate(&daos.AuditLog{}); err != nil {
		t.Fatalf("migrate audit database: %v", err)
	}
	previousDB := config.DBConnect
	config.DBConnect = db
	t.Cleanup(func() { config.DBConnect = previousDB })

	container := restful.NewContainer()
	ws := new(restful.WebService).Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)
	ws.Route(ws.PUT("/api/v1/channels/{id}").
		Filter(func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
			req.SetAttribute(config.RequestUserId, "admin-1")
			chain.ProcessFilter(req, resp)
		}).Filter(Audit).To(func(_ *restful.Request, resp *restful.Response) {
		resp.WriteHeader(http.StatusOK)
	}))
	container.Add(ws)

	request := httptest.NewRequest(http.MethodPut, "/api/v1/channels/channel-1", strings.NewReader(`{"apiKey":"must-not-appear"}`))
	request.Header.Set("Content-Type", restful.MIME_JSON)
	request.RemoteAddr = "192.0.2.10:1234"
	recorder := httptest.NewRecorder()
	container.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected response status: %d", recorder.Code)
	}

	var record daos.AuditLog
	if err = db.First(&record).Error; err != nil {
		t.Fatalf("read audit record: %v", err)
	}
	if record.OperatorID != "admin-1" || record.Action != "update" || record.ResourceType != "channels" || record.ResourceID != "channel-1" || record.RemoteIP != "192.0.2.10" {
		t.Fatalf("unexpected audit record: %#v", record)
	}
	if strings.Contains(record.Summary, "must-not-appear") {
		t.Fatal("audit summary leaked the request body")
	}
}

func TestAuditRemoteIP(t *testing.T) {
	tests := []struct {
		name          string
		realIP        string
		forwardedFor  string
		remoteAddress string
		want          string
	}{
		{
			name:          "prefer X-Real-Ip",
			realIP:        "203.0.113.10",
			forwardedFor:  "198.51.100.20",
			remoteAddress: "192.0.2.30:1234",
			want:          "203.0.113.10",
		},
		{
			name:          "fall back to X-Forwarded-For",
			forwardedFor:  "198.51.100.20, 192.0.2.30",
			remoteAddress: "192.0.2.30:1234",
			want:          "198.51.100.20, 192.0.2.30",
		},
		{
			name:          "fall back to RemoteAddr",
			remoteAddress: "192.0.2.30:1234",
			want:          "192.0.2.30",
		},
		{
			name:          "normalize IPv6 loopback",
			remoteAddress: "[::1]:1234",
			want:          "127.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpRequest := httptest.NewRequest(http.MethodPut, "/api/v1/channels/channel-1", nil)
			httpRequest.RemoteAddr = tt.remoteAddress
			httpRequest.Header.Set("X-Real-Ip", tt.realIP)
			httpRequest.Header.Set("X-Forwarded-For", tt.forwardedFor)

			if got := auditRemoteIP(restful.NewRequest(httpRequest)); got != tt.want {
				t.Fatalf("auditRemoteIP() = %q, want %q", got, tt.want)
			}
		})
	}
}
