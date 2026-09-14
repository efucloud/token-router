package v1

import (
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/efucloud/token-router/pkg/apis/filters"
	"github.com/efucloud/token-router/pkg/config"
	"github.com/efucloud/token-router/pkg/services"
	"github.com/efucloud/token-router/pkg/utils"
	restful "github.com/emicklei/go-restful/v3"
)

type GatewayResource struct {
	Svc services.GatewayService
}

type openAIErrorBody struct {
	Error openAIError `json:"error"`
}

type openAIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

func (r GatewayResource) AddWebService(ws *restful.WebService) {
	ws.Route(ws.GET("/v1/models").To(r.models))
	ws.Route(ws.POST("/v1/chat/completions").To(r.chat).
		Produces(restful.MIME_JSON, "application/x-ndjson"))
	ws.Route(ws.POST("/v1/responses").To(r.responses).
		Produces(restful.MIME_JSON, "text/event-stream"))
	ws.Route(ws.POST("/v1/embeddings").To(r.embeddings))
}

func writeOpenAIError(resp *restful.Response, status int, code, message string) {
	resp.Header().Set("Content-Type", "application/json")
	_ = resp.WriteHeaderAndEntity(status, openAIErrorBody{Error: openAIError{Message: message, Type: "gateway_error", Code: code}})
}

func (r GatewayResource) principal(req *restful.Request, resp *restful.Response) (services.GatewayPrincipal, bool) {
	token := filters.GetRequestToken(config.AuthHeader, req)
	if token == "" {
		writeOpenAIError(resp, http.StatusUnauthorized, "invalid_api_key", "missing API key or system token")
		return services.GatewayPrincipal{}, false
	}
	var principal services.GatewayPrincipal
	var err error
	isAPIKey := strings.HasPrefix(token, utils.APITokenPrefix)
	if isAPIKey {
		principal, err = r.Svc.Authenticate(req.Request.Context(), token)
	} else {
		accountID := filters.GetAccountIdFromToken(req)
		principal, err = r.Svc.AuthenticateSystemToken(req.Request.Context(), accountID)
	}
	if err != nil {
		if isAPIKey {
			writeOpenAIError(resp, http.StatusUnauthorized, "invalid_api_key", "invalid or inactive API key")
		} else {
			writeOpenAIError(resp, http.StatusUnauthorized, "invalid_token", "invalid or inactive system token")
		}
		return services.GatewayPrincipal{}, false
	}
	if err = r.Svc.AuthorizeIP(principal, req.Request.RemoteAddr); err != nil {
		writeOpenAIError(resp, http.StatusForbidden, "ip_not_allowed", "request IP is not allowed by this API key")
		return services.GatewayPrincipal{}, false
	}
	return principal, true
}

func writeQuotaError(resp *restful.Response, quotaError services.GatewayQuotaError) {
	if quotaError.Limit > 0 {
		resp.Header().Set("X-RateLimit-Limit", strconv.FormatInt(quotaError.Limit, 10))
	}
	if !quotaError.ResetAt.IsZero() {
		seconds := int64(math.Ceil(time.Until(quotaError.ResetAt).Seconds()))
		if seconds < 1 {
			seconds = 1
		}
		resp.Header().Set("Retry-After", strconv.FormatInt(seconds, 10))
		resp.Header().Set("X-RateLimit-Reset", strconv.FormatInt(quotaError.ResetAt.Unix(), 10))
	}
	writeOpenAIError(resp, http.StatusTooManyRequests, "quota_exceeded", "user or API key quota is exhausted")
}

func (r GatewayResource) models(req *restful.Request, resp *restful.Response) {
	principal, ok := r.principal(req, resp)
	if !ok {
		return
	}
	models, err := r.Svc.PublishedModels(req.Request.Context(), principal)
	if err != nil {
		writeOpenAIError(resp, http.StatusInternalServerError, "gateway_error", "failed to load published models")
		return
	}
	data := make([]map[string]any, 0, len(models))
	createdAt := time.Now().Unix()
	for _, model := range models {
		capabilities := make([]string, 0)
		_ = json.Unmarshal([]byte(model.Capabilities), &capabilities)
		data = append(data, map[string]any{
			"id": model.Name, "object": "model", "created": createdAt, "owned_by": "token-router",
			"display_name": model.DisplayName, "description": model.Description, "modality": model.Modality,
			"context_window": model.ContextWindow, "max_output_tokens": model.MaxOutputTokens,
			"capabilities": capabilities,
		})
	}
	_ = resp.WriteEntity(map[string]any{"object": "list", "data": data})
}

func (r GatewayResource) chat(req *restful.Request, resp *restful.Response) {
	r.proxy(req, resp, "/chat/completions", "chat")
}

func (r GatewayResource) responses(req *restful.Request, resp *restful.Response) {
	r.proxy(req, resp, "/responses", "chat")
}

func (r GatewayResource) embeddings(req *restful.Request, resp *restful.Response) {
	r.proxy(req, resp, "/embeddings", "embedding")
}

func (r GatewayResource) proxy(req *restful.Request, resp *restful.Response, endpoint, modality string) {
	principal, ok := r.principal(req, resp)
	if !ok {
		return
	}
	limit := config.ApplicationConfig.Gateway.MaxRequestBodyBytes
	body, err := io.ReadAll(io.LimitReader(req.Request.Body, limit+1))
	if err != nil || int64(len(body)) > limit {
		writeOpenAIError(resp, http.StatusRequestEntityTooLarge, "request_too_large", "request body is too large")
		return
	}
	var input struct {
		Model      string `json:"model"`
		Stream     bool   `json:"stream"`
		Background bool   `json:"background"`
	}
	if json.Unmarshal(body, &input) != nil || strings.TrimSpace(input.Model) == "" {
		writeOpenAIError(resp, http.StatusBadRequest, "invalid_request", "model is required")
		return
	}
	if endpoint == "/responses" && input.Background {
		writeOpenAIError(resp, http.StatusBadRequest, "unsupported_parameter", "background responses are not supported")
		return
	}
	if endpoint == "/chat/completions" {
		body, err = r.Svc.InjectConversationTools(
			req.Request.Context(), principal,
			req.HeaderParameter(services.GatewayConversationHeader), body,
		)
		if err != nil {
			writeOpenAIError(resp, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
	}
	model, routes, err := r.Svc.ResolveModelAndRoutes(req.Request.Context(), principal, input.Model, modality)
	if err != nil {
		writeOpenAIError(resp, http.StatusNotFound, "model_not_found", err.Error())
		return
	}
	err = r.Svc.Proxy(req.Request.Context(), resp.ResponseWriter, req.Request, principal, model, routes, endpoint, body, input.Stream)
	if err == nil {
		return
	}
	var quotaError services.GatewayQuotaError
	if errors.As(err, &quotaError) {
		writeQuotaError(resp, quotaError)
		return
	}
	writeOpenAIError(resp, http.StatusBadGateway, "upstream_error", "all upstream routes failed")
}
