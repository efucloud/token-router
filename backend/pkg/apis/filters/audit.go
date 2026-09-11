package filters

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"github.com/efucloud/token-router/pkg/config"
	"github.com/efucloud/token-router/pkg/models/daos"
	"github.com/efucloud/token-router/pkg/utils"
	restful "github.com/emicklei/go-restful/v3"
)

func auditAction(method, routePath string) string {
	segments := strings.Split(strings.Trim(routePath, "/"), "/")
	if len(segments) > 0 {
		last := segments[len(segments)-1]
		if !strings.HasPrefix(last, "{") && last != "account" && last != "models" && last != "providers" && last != "channels" && last != "routes" {
			return last
		}
	}
	switch method {
	case http.MethodPost:
		return "create"
	case http.MethodPut, http.MethodPatch:
		return "update"
	case http.MethodDelete:
		return "delete"
	default:
		return strings.ToLower(method)
	}
}

func auditResource(routePath string) string {
	path := strings.TrimPrefix(routePath, config.APIPrefix)
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) == 0 || segments[0] == "" {
		return "unknown"
	}
	return segments[0]
}

func auditRemoteIP(req *restful.Request) string {
	remoteAddress := req.HeaderParameter("X-Real-Ip")
	if remoteAddress == "" {
		remoteAddress = req.HeaderParameter("X-Forwarded-For")
	}
	if remoteAddress == "" {
		remoteAddress, _, _ = net.SplitHostPort(req.Request.RemoteAddr)
		if remoteAddress == "::1" {
			remoteAddress = "127.0.0.1"
		}
	}
	return remoteAddress
}

// Audit records authorized control-plane mutations without reading request
// bodies. This intentionally prevents credentials and other sensitive fields
// from entering the audit trail.
func Audit(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	method := req.Request.Method
	if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
		chain.ProcessFilter(req, resp)
		return
	}
	requestID := utils.GenerateDatabaseId()
	resp.Header().Set("X-Request-Id", requestID)
	chain.ProcessFilter(req, resp)

	operatorID, _ := req.Attribute(config.RequestUserId).(string)
	routePath := req.SelectedRoutePath()
	statusCode := resp.StatusCode()
	result := "failed"
	if statusCode >= 200 && statusCode < 400 {
		result = "succeeded"
	}
	summaryFields := map[string]string{
		"route": routePath,
		"note":  "request body omitted to protect credentials and sensitive data",
	}
	if changedFields, ok := req.Attribute("auditChangedFields").(string); ok && changedFields != "" {
		summaryFields["changedFields"] = changedFields
	}
	summary, _ := json.Marshal(summaryFields)
	record := daos.AuditLog{
		GatewayRecord: daos.GatewayRecord{ID: utils.GenerateDatabaseId()},
		OperatorID:    operatorID, Action: auditAction(method, routePath), ResourceType: auditResource(routePath),
		ResourceID: req.PathParameter("id"), Method: method, RoutePath: routePath, Summary: string(summary),
		Result: result, StatusCode: statusCode, RequestID: requestID, RemoteIP: auditRemoteIP(req),
	}
	if err := config.DBConnect.Create(&record).Error; err != nil {
		config.Logger.Errorf("write control-plane audit log failed: %v", err)
	}
}
