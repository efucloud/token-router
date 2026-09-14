package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/efucloud/token-router/pkg/config"
	"github.com/efucloud/token-router/pkg/models/daos"
	"gorm.io/gorm"
)

func createGatewayPrincipal(t *testing.T, db *gorm.DB, token daos.APIToken) GatewayPrincipal {
	t.Helper()
	account := daos.Account{ID: token.AccountID, Username: token.AccountID, Enable: true}
	if err := db.Create(&account).Error; err != nil {
		t.Fatalf("create account: %v", err)
	}
	if token.AllowedModels == "" {
		token.AllowedModels = "[]"
	}
	if token.IPAllowlist == "" {
		token.IPAllowlist = "[]"
	}
	if err := db.Create(&token).Error; err != nil {
		t.Fatalf("create token: %v", err)
	}
	return GatewayPrincipal{Account: account, Token: token}
}

func TestResponsesRequestRewriteAndUsage(t *testing.T) {
	body, err := rewriteGatewayRequest([]byte(`{"model":"unified","input":"hello","stream":true,"tools":[{"type":"function","name":"lookup"}]}`), "upstream-model", "/responses", true)
	if err != nil {
		t.Fatalf("rewrite responses request: %v", err)
	}
	var payload map[string]any
	if err = json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode rewritten request: %v", err)
	}
	if payload["model"] != "upstream-model" {
		t.Fatalf("model was not rewritten: %#v", payload)
	}
	if _, exists := payload["stream_options"]; exists {
		t.Fatal("responses requests must not receive chat-completions stream_options")
	}
	if tools, ok := payload["tools"].([]any); !ok || len(tools) != 1 {
		t.Fatalf("tools were not preserved: %#v", payload["tools"])
	}

	usage, ok := parseGatewayUsage([]byte(`{"usage":{"input_tokens":11,"output_tokens":7,"total_tokens":18}}`))
	if !ok || usage.PromptTokens != 11 || usage.CompletionTokens != 7 || usage.TotalTokens != 18 {
		t.Fatalf("unexpected responses usage: %#v, found=%v", usage, ok)
	}
	usage, ok = parseGatewayUsage([]byte(`{"type":"response.completed","response":{"usage":{"input_tokens":5,"output_tokens":3,"total_tokens":8}}}`))
	if !ok || usage.TotalTokens != 8 {
		t.Fatalf("unexpected streaming responses usage: %#v, found=%v", usage, ok)
	}
}

func TestResponsesTokenEstimateUsesMaxOutputTokens(t *testing.T) {
	estimate := estimateGatewayTokens([]byte(`{"model":"gpt","input":"hello","max_output_tokens":4096}`), 10000)
	if estimate < 4096 {
		t.Fatalf("expected max_output_tokens in reservation estimate, got %d", estimate)
	}
}

func TestInjectConversationToolsBuildsDefinitionsOnServer(t *testing.T) {
	db := dashboardTestDatabase(t)
	previousConfig := config.ApplicationConfig
	config.ApplicationConfig = &config.Config{Chat: config.ChatConfig{BuiltinTools: config.ChatBuiltinToolsConfig{
		Enabled: true, WorkspaceDirectory: t.TempDir(),
	}}}
	config.ApplicationConfig.Chat.Default()
	t.Cleanup(func() { config.ApplicationConfig = previousConfig })

	principal := createGatewayPrincipal(t, db, daos.APIToken{
		GatewayRecord: daos.GatewayRecord{ID: "tool-injection-token"}, AccountID: "tool-injection-user", Name: "tools",
		KeyHash: "tool-injection-hash", KeyPrefix: "tr_tool_injection", Status: "active",
	})
	conversation := daos.ChatConversation{
		GatewayRecord: daos.GatewayRecord{ID: "tool-injection-conversation"},
		AccountID:     principal.Account.ID, Model: "unified-model", Skills: "[]",
		MCPServers: `["builtin"]`, MCPServersInitialized: true,
	}
	if err := db.Create(&conversation).Error; err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	body, err := (GatewayService{}).InjectConversationTools(
		context.Background(), principal, conversation.ID,
		[]byte(`{"model":"unified-model","messages":[],"tool_choice":"auto","tools":[{"type":"function","function":{"name":"client_tool"}}]}`),
	)
	if err != nil {
		t.Fatalf("inject conversation tools: %v", err)
	}
	var payload map[string]any
	if err = json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode injected request: %v", err)
	}
	if payload["tool_choice"] != "auto" {
		t.Fatalf("tool choice was not preserved: %#v", payload)
	}
	tools, ok := payload["tools"].([]any)
	if !ok || len(tools) == 0 {
		t.Fatalf("server tools were not injected: %#v", payload["tools"])
	}
	for _, item := range tools {
		definition, _ := item.(map[string]any)
		function, _ := definition["function"].(map[string]any)
		name, _ := function["name"].(string)
		if name == "client_tool" || !strings.HasPrefix(name, "mcp__builtin__") {
			t.Fatalf("unexpected injected tool name %q", name)
		}
	}
}

func TestInjectConversationToolsRequiresToolChoice(t *testing.T) {
	db := dashboardTestDatabase(t)
	principal := createGatewayPrincipal(t, db, daos.APIToken{
		GatewayRecord: daos.GatewayRecord{ID: "no-choice-token"}, AccountID: "no-choice-user", Name: "no choice",
		KeyHash: "no-choice-hash", KeyPrefix: "tr_no_choice", Status: "active",
	})
	conversation := daos.ChatConversation{
		GatewayRecord: daos.GatewayRecord{ID: "no-choice-conversation"},
		AccountID:     principal.Account.ID, Model: "unified-model", Skills: "[]", MCPServers: "[]", MCPServersInitialized: true,
	}
	if err := db.Create(&conversation).Error; err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	body, err := (GatewayService{}).InjectConversationTools(
		context.Background(), principal, conversation.ID,
		[]byte(`{"model":"unified-model","messages":[],"tools":[{"type":"function"}]}`),
	)
	if err != nil {
		t.Fatalf("prepare request without tool choice: %v", err)
	}
	var payload map[string]any
	if err = json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode prepared request: %v", err)
	}
	if _, exists := payload["tools"]; exists {
		t.Fatalf("client-provided tools were not removed: %#v", payload)
	}
}

func TestResponsesProxySupportsJSONAndStreaming(t *testing.T) {
	db := dashboardTestDatabase(t)
	previousConfig := config.ApplicationConfig
	config.ApplicationConfig = &config.Config{Gateway: config.GatewayConfig{
		MaxReservedTokens: 4096, MaxAttempts: 1, UpstreamTimeoutSeconds: 5,
		MaxResponseBodyBytes: 1 << 20, FailureThreshold: 3, CooldownSeconds: 60,
	}}
	t.Cleanup(func() { config.ApplicationConfig = previousConfig })
	principal := createGatewayPrincipal(t, db, daos.APIToken{
		GatewayRecord: daos.GatewayRecord{ID: "responses-token"}, AccountID: "responses-user", Name: "responses",
		KeyHash: "responses-hash", KeyPrefix: "tr_responses", Status: "active",
	})
	channel := daos.Channel{GatewayRecord: daos.GatewayRecord{ID: "responses-channel"}, ProviderID: "provider", Name: "channel", BaseURL: "unused", Weight: 1, Status: "enabled", HealthStatus: "healthy"}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("create channel: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/v1/responses" {
			http.NotFound(resp, req)
			return
		}
		var payload map[string]any
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil || payload["model"] != "upstream-model" {
			http.Error(resp, "invalid model", http.StatusBadRequest)
			return
		}
		if tools, ok := payload["tools"].([]any); !ok || len(tools) != 1 {
			http.Error(resp, "missing tools", http.StatusBadRequest)
			return
		}
		resp.Header().Set("Content-Type", "application/json")
		if payload["stream"] == true {
			resp.Header().Set("Content-Type", "text/event-stream")
			_, _ = resp.Write([]byte("event: response.completed\n"))
			_, _ = resp.Write([]byte("data: {\"type\":\"response.completed\",\"response\":{\"model\":\"upstream-model\",\"usage\":{\"input_tokens\":3,\"output_tokens\":2,\"total_tokens\":5}}}\n\n"))
			return
		}
		_, _ = resp.Write([]byte(`{"id":"resp_1","model":"upstream-model","output":[],"usage":{"input_tokens":3,"output_tokens":2,"total_tokens":5}}`))
	}))
	defer server.Close()
	channel.BaseURL = server.URL + "/v1"
	if err := db.Model(&channel).Update("base_url", channel.BaseURL).Error; err != nil {
		t.Fatalf("update channel URL: %v", err)
	}

	service := GatewayService{}
	model := GatewayModel{ID: "responses-model", Name: "unified-model", Modality: "chat"}
	route := GatewayRoute{RouteID: "responses-route", ChannelID: channel.ID, UpstreamModel: "upstream-model", BaseURL: channel.BaseURL, Weight: 1, TimeoutSeconds: 5}
	for _, streaming := range []bool{false, true} {
		body := []byte(`{"model":"unified-model","input":"hello","tools":[{"type":"function","name":"lookup"}],"stream":` + fmt.Sprint(streaming) + `}`)
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
		if err := service.Proxy(context.Background(), recorder, request, principal, model, []GatewayRoute{route}, "/responses", body, streaming); err != nil {
			t.Fatalf("proxy responses stream=%v: %v", streaming, err)
		}
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"model":"unified-model"`) {
			t.Fatalf("unexpected normalized response stream=%v: status=%d body=%s", streaming, recorder.Code, recorder.Body.String())
		}
	}
}

func TestChatCompletionsStreamingUsesNDJSON(t *testing.T) {
	db := dashboardTestDatabase(t)
	previousConfig := config.ApplicationConfig
	config.ApplicationConfig = &config.Config{Gateway: config.GatewayConfig{
		MaxReservedTokens: 4096, MaxAttempts: 1, UpstreamTimeoutSeconds: 5,
		MaxResponseBodyBytes: 1 << 20, FailureThreshold: 3, CooldownSeconds: 60,
	}}
	t.Cleanup(func() { config.ApplicationConfig = previousConfig })
	principal := createGatewayPrincipal(t, db, daos.APIToken{
		GatewayRecord: daos.GatewayRecord{ID: "chat-stream-token"}, AccountID: "chat-stream-user", Name: "chat stream",
		KeyHash: "chat-stream-hash", KeyPrefix: "tr_chat_stream", Status: "active",
	})
	channel := daos.Channel{GatewayRecord: daos.GatewayRecord{ID: "chat-stream-channel"}, ProviderID: "provider", Name: "channel", BaseURL: "unused", Weight: 1, Status: "enabled", HealthStatus: "healthy"}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("create channel: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/v1/chat/completions" {
			http.NotFound(resp, req)
			return
		}
		if req.Header.Get("Accept") != "text/event-stream" {
			http.Error(resp, "gateway must request an upstream event stream", http.StatusBadRequest)
			return
		}
		var payload map[string]any
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil || payload["model"] != "upstream-model" || payload["stream"] != true {
			http.Error(resp, "invalid streaming request", http.StatusBadRequest)
			return
		}
		resp.Header().Set("Content-Type", "text/event-stream")
		_, _ = resp.Write([]byte("event: message\n"))
		_, _ = resp.Write([]byte("data: {\"id\":\"chunk_1\",\"model\":\"upstream-model\",\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\n"))
		_, _ = resp.Write([]byte("data: {\"id\":\"chunk_1\",\"model\":\"upstream-model\",\"choices\":[],\"usage\":{\"prompt_tokens\":3,\"completion_tokens\":2,\"total_tokens\":5}}\n\n"))
		_, _ = resp.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()
	channel.BaseURL = server.URL + "/v1"
	if err := db.Model(&channel).Update("base_url", channel.BaseURL).Error; err != nil {
		t.Fatalf("update channel URL: %v", err)
	}

	body := []byte(`{"model":"unified-model","messages":[{"role":"user","content":"hello"}],"stream":true}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	request.Header.Set("Accept", "application/x-ndjson")
	model := GatewayModel{ID: "chat-stream-model", Name: "unified-model", Modality: "chat"}
	route := GatewayRoute{RouteID: "chat-stream-route", ChannelID: channel.ID, UpstreamModel: "upstream-model", BaseURL: channel.BaseURL, Weight: 1, TimeoutSeconds: 5}
	if err := (GatewayService{}).Proxy(context.Background(), recorder, request, principal, model, []GatewayRoute{route}, "/chat/completions", body, true); err != nil {
		t.Fatalf("proxy streaming chat completion: %v", err)
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/x-ndjson") {
		t.Fatalf("expected NDJSON response, got %q", contentType)
	}
	if !recorder.Flushed {
		t.Fatal("streaming response was not flushed")
	}
	lines := strings.Split(strings.TrimSpace(recorder.Body.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected two NDJSON chunks, got %d: %q", len(lines), recorder.Body.String())
	}
	for _, line := range lines {
		if !json.Valid([]byte(line)) || strings.HasPrefix(line, "data:") || line == "[DONE]" {
			t.Fatalf("invalid NDJSON line: %q", line)
		}
		if !strings.Contains(line, `"model":"unified-model"`) {
			t.Fatalf("unified model was not restored: %q", line)
		}
	}
}

func requireGatewayQuotaError(t *testing.T, err error, dimension string) GatewayQuotaError {
	t.Helper()
	var quotaError GatewayQuotaError
	if !errors.As(err, &quotaError) {
		t.Fatalf("expected quota error, got %v", err)
	}
	if quotaError.Dimension != dimension {
		t.Fatalf("expected %s quota, got %#v", dimension, quotaError)
	}
	return quotaError
}

func TestGatewayReservationAndSettlement(t *testing.T) {
	db := dashboardTestDatabase(t)
	previousConfig := config.ApplicationConfig
	config.ApplicationConfig = &config.Config{Gateway: config.GatewayConfig{MaxReservedTokens: 1000}}
	t.Cleanup(func() { config.ApplicationConfig = previousConfig })

	account := daos.Account{ID: "u1", Username: "alice", Enable: true, TokenLimit: 100, RequestLimit: 2}
	token := daos.APIToken{GatewayRecord: daos.GatewayRecord{ID: "t1"}, AccountID: "u1", Name: "test", KeyHash: "hash", KeyPrefix: "tr_test", Status: "active", AllowedModels: "[]", TokenLimit: 50, RequestLimit: 1, IPAllowlist: "[]"}
	if err := db.Create(&account).Error; err != nil {
		t.Fatalf("create account: %v", err)
	}
	if err := db.Create(&token).Error; err != nil {
		t.Fatalf("create token: %v", err)
	}
	principal := GatewayPrincipal{Account: account, Token: token}
	model := GatewayModel{ID: "m1", Name: "model"}
	reservation, err := (GatewayService{}).Reserve(context.Background(), principal, model, "chat/completions", false, 30)
	if err != nil {
		t.Fatalf("reserve quota: %v", err)
	}
	var reservedToken daos.APIToken
	if err = db.First(&reservedToken, "id = ?", "t1").Error; err != nil {
		t.Fatalf("read reserved token: %v", err)
	}
	if reservedToken.ReservedTokens != 30 || reservedToken.ReservedRequests != 1 {
		t.Fatalf("quota was not reserved: %#v", reservedToken)
	}
	if err = (GatewayService{}).Settle(context.Background(), reservation, GatewayUsage{PromptTokens: 7, CompletionTokens: 3, TotalTokens: 10}, true, false, 200, "", "", "c1"); err != nil {
		t.Fatalf("settle quota: %v", err)
	}
	if err = db.First(&reservedToken, "id = ?", "t1").Error; err != nil {
		t.Fatalf("read settled token: %v", err)
	}
	if reservedToken.ReservedTokens != 0 || reservedToken.ReservedRequests != 0 || reservedToken.UsedTokens != 10 || reservedToken.UsedRequests != 1 {
		t.Fatalf("quota was not settled: %#v", reservedToken)
	}
	if err = (GatewayService{}).Settle(context.Background(), reservation, GatewayUsage{TotalTokens: 10}, true, false, 200, "", "", "c1"); err != nil {
		t.Fatalf("repeat settlement: %v", err)
	}
	if err = db.First(&reservedToken, "id = ?", "t1").Error; err != nil {
		t.Fatalf("read idempotently settled token: %v", err)
	}
	if reservedToken.UsedTokens != 10 || reservedToken.UsedRequests != 1 {
		t.Fatalf("settlement was not idempotent: %#v", reservedToken)
	}
}

func TestSystemTokenReservationAndSettlement(t *testing.T) {
	db := dashboardTestDatabase(t)
	account := daos.Account{ID: "system-user", Username: "system-user", Enable: true, TokenLimit: 100, RequestLimit: 2}
	if err := db.Create(&account).Error; err != nil {
		t.Fatalf("create account: %v", err)
	}
	principal := GatewayPrincipal{Account: account, SystemToken: true}
	model := GatewayModel{ID: "m1", Name: "model"}
	service := GatewayService{}
	reservation, err := service.Reserve(context.Background(), principal, model, "chat/completions", true, 30)
	if err != nil {
		t.Fatalf("reserve system token quota: %v", err)
	}
	if reservation.APITokenID != "" {
		t.Fatalf("system token reservation must not be attached to an API key: %#v", reservation)
	}
	var reservedAccount daos.Account
	if err = db.First(&reservedAccount, "id = ?", account.ID).Error; err != nil {
		t.Fatalf("read reserved account: %v", err)
	}
	if reservedAccount.ReservedTokens != 30 || reservedAccount.ReservedRequests != 1 {
		t.Fatalf("account quota was not reserved: %#v", reservedAccount)
	}
	var log daos.UsageLog
	if err = db.First(&log, "request_id = ?", reservation.RequestID).Error; err != nil {
		t.Fatalf("read usage log: %v", err)
	}
	if log.APITokenID != systemTokenUsageID || log.TokenPrefix != systemTokenUsageID {
		t.Fatalf("system token usage was not identified: %#v", log)
	}
	if err = service.Settle(context.Background(), reservation, GatewayUsage{PromptTokens: 7, CompletionTokens: 3, TotalTokens: 10}, true, false, 200, "", "", "c1"); err != nil {
		t.Fatalf("settle system token quota: %v", err)
	}
	if err = db.First(&reservedAccount, "id = ?", account.ID).Error; err != nil {
		t.Fatalf("read settled account: %v", err)
	}
	if reservedAccount.ReservedTokens != 0 || reservedAccount.ReservedRequests != 0 || reservedAccount.UsedTokens != 10 || reservedAccount.UsedRequests != 1 {
		t.Fatalf("account quota was not settled: %#v", reservedAccount)
	}
}

func TestGatewayAuthorizeIP(t *testing.T) {
	service := GatewayService{}
	principal := GatewayPrincipal{Token: daos.APIToken{IPAllowlist: `["192.0.2.10","10.0.0.0/8","2001:db8::/32"]`}}
	for _, remoteAddress := range []string{"192.0.2.10:443", "10.2.3.4:1234", "[2001:db8::1]:443"} {
		if err := service.AuthorizeIP(principal, remoteAddress); err != nil {
			t.Fatalf("expected %s to be allowed: %v", remoteAddress, err)
		}
	}
	if err := service.AuthorizeIP(principal, "203.0.113.8:443"); err == nil {
		t.Fatal("expected address outside the allowlist to be rejected")
	}
	principal.Token.IPAllowlist = "[]"
	if err := service.AuthorizeIP(principal, "not-an-address"); err != nil {
		t.Fatalf("an empty allowlist must not restrict clients: %v", err)
	}
}

func TestGatewayRPMAndTPMLimits(t *testing.T) {
	db := dashboardTestDatabase(t)
	previousConfig := config.ApplicationConfig
	config.ApplicationConfig = &config.Config{Gateway: config.GatewayConfig{MaxReservedTokens: 1000}}
	t.Cleanup(func() { config.ApplicationConfig = previousConfig })

	principal := createGatewayPrincipal(t, db, daos.APIToken{
		GatewayRecord: daos.GatewayRecord{ID: "rate-token"}, AccountID: "rate-user", Name: "rate",
		KeyHash: "rate-hash", KeyPrefix: "tr_rate", Status: "active", RPM: 1, TPM: 50,
	})
	model := GatewayModel{ID: "m1", Name: "model"}
	service := GatewayService{}
	first, err := service.Reserve(context.Background(), principal, model, "chat/completions", false, 30)
	if err != nil {
		t.Fatalf("reserve first request: %v", err)
	}
	rpmError := requireGatewayQuotaError(t, func() error {
		_, reserveErr := service.Reserve(context.Background(), principal, model, "chat/completions", false, 1)
		return reserveErr
	}(), "API key RPM")
	if rpmError.Limit != 1 || rpmError.ResetAt.IsZero() {
		t.Fatalf("unexpected RPM metadata: %#v", rpmError)
	}
	if err = service.Settle(context.Background(), first, GatewayUsage{TotalTokens: 30}, true, false, 200, "", "", "c1"); err != nil {
		t.Fatalf("settle first request: %v", err)
	}

	if err = db.Model(&daos.APIRateLimitBucket{}).Where("api_token_id = ?", principal.Token.ID).Update("used_requests", 0).Error; err != nil {
		t.Fatalf("reset RPM bucket for TPM assertion: %v", err)
	}
	requireGatewayQuotaError(t, func() error {
		_, reserveErr := service.Reserve(context.Background(), principal, model, "chat/completions", false, 21)
		return reserveErr
	}(), "API key TPM")
}

func TestGatewayConcurrencyLeaseReleasedOnSettlement(t *testing.T) {
	db := dashboardTestDatabase(t)
	previousConfig := config.ApplicationConfig
	config.ApplicationConfig = &config.Config{Gateway: config.GatewayConfig{MaxReservedTokens: 1000, MaxAttempts: 2, UpstreamTimeoutSeconds: 30}}
	t.Cleanup(func() { config.ApplicationConfig = previousConfig })

	principal := createGatewayPrincipal(t, db, daos.APIToken{
		GatewayRecord: daos.GatewayRecord{ID: "concurrency-token"}, AccountID: "concurrency-user", Name: "concurrency",
		KeyHash: "concurrency-hash", KeyPrefix: "tr_concurrency", Status: "active", MaxConcurrency: 1,
	})
	model := GatewayModel{ID: "m1", Name: "model"}
	service := GatewayService{}
	first, err := service.Reserve(context.Background(), principal, model, "chat/completions", false, 10)
	if err != nil {
		t.Fatalf("reserve first request: %v", err)
	}
	requireGatewayQuotaError(t, func() error {
		_, reserveErr := service.Reserve(context.Background(), principal, model, "chat/completions", false, 10)
		return reserveErr
	}(), "API key concurrency")
	if err = service.Settle(context.Background(), first, GatewayUsage{TotalTokens: 1}, true, false, 200, "", "", "c1"); err != nil {
		t.Fatalf("settle first request: %v", err)
	}
	if _, err = service.Reserve(context.Background(), principal, model, "chat/completions", false, 10); err != nil {
		t.Fatalf("expected released concurrency slot to be reusable: %v", err)
	}
}
