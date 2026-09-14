package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/efucloud/token-router/pkg/config"
	"github.com/efucloud/token-router/pkg/models/daos"
	"github.com/efucloud/token-router/pkg/models/dtos"
)

func TestControlPlaneCRUDAndChannelHealth(t *testing.T) {
	dashboardTestDatabase(t)
	previousConfig := config.ApplicationConfig
	config.ApplicationConfig = &config.Config{Gateway: config.GatewayConfig{UpstreamTimeoutSeconds: 5}}
	t.Cleanup(func() { config.ApplicationConfig = previousConfig })

	server := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/v1/models" {
			http.NotFound(resp, req)
			return
		}
		if req.Header.Get("Authorization") != "Bearer secret" {
			http.Error(resp, "unauthorized", http.StatusUnauthorized)
			return
		}
		resp.Header().Set("Content-Type", "application/json")
		_, _ = resp.Write([]byte(`{"object":"list","data":[{"id":"qwen-coder"}]}`))
	}))
	defer server.Close()

	ctx := context.Background()
	svc := ControlPlaneService{}
	provider, errorData := svc.CreateProvider(ctx, dtos.ProviderInput{Name: "Bailian", Type: "openai-compatible", Status: "enabled", Config: map[string]any{}})
	if errorData.IsNotNil() {
		t.Fatalf("create provider: %v", errorData.Err)
	}
	channel, errorData := svc.CreateChannel(ctx, dtos.ChannelInput{ProviderID: provider.ID, Name: "primary", BaseURL: server.URL + "/v1", APIKey: "secret", Weight: 1, Status: "enabled"})
	if errorData.IsNotNil() || !channel.CredentialConfigured {
		t.Fatalf("create channel: %#v, %v", channel, errorData.Err)
	}
	model, errorData := svc.CreateModel(ctx, dtos.AIModelInput{Name: "coder", DisplayName: "Coder", Modality: "chat", Capabilities: []string{"streaming"}, Status: "active"})
	if errorData.IsNotNil() {
		t.Fatalf("create model: %v", errorData.Err)
	}
	modelUpdate := dtos.AIModelInput{Name: "coder", DisplayName: "Coder", Modality: "chat", Capabilities: []string{"streaming"}, Status: "active", Version: model.Version}
	updatedModel, errorData := svc.UpdateModel(ctx, model.ID, modelUpdate)
	if errorData.IsNotNil() || updatedModel.Version != model.Version+1 {
		t.Fatalf("update model with version: %#v, %v", updatedModel, errorData.Err)
	}
	if _, staleError := svc.UpdateModel(ctx, model.ID, modelUpdate); staleError.IsNil() || staleError.ResponseCode != http.StatusConflict {
		t.Fatalf("stale model update must conflict: %#v", staleError)
	}
	model = updatedModel
	route, errorData := svc.CreateRoute(ctx, dtos.ModelRouteInput{ModelID: model.ID, ChannelID: channel.ID, UpstreamModel: "qwen-coder", Weight: 1, Status: "enabled"})
	if errorData.IsNotNil() || route.ProviderName != "Bailian" {
		t.Fatalf("create route: %#v, %v", route, errorData.Err)
	}

	testResult, errorData := svc.TestChannel(ctx, channel.ID)
	if errorData.IsNotNil() || !testResult.Success || testResult.ModelCount != 1 || testResult.Models[0] != "qwen-coder" {
		t.Fatalf("test channel: %#v, %v", testResult, errorData.Err)
	}
	importResult, errorData := svc.ImportChannelModels(ctx, channel.ID, dtos.ChannelModelImportInput{Models: testResult.Models})
	if errorData.IsNotNil() || importResult.ModelsCreated != 1 || importResult.RoutesCreated != 1 || importResult.Items[0].ModelName != "qwen-coder" {
		t.Fatalf("import discovered model: %#v, %v", importResult, errorData.Err)
	}
	var importedModel daos.AIModel
	if err := config.DBConnect.Where("name = ?", "qwen-coder").First(&importedModel).Error; err != nil || importedModel.Status != "active" {
		t.Fatalf("imported model must be active by default: %#v, %v", importedModel, err)
	}
	var importedRoute daos.ModelRoute
	if err := config.DBConnect.Where("model_id = ? AND channel_id = ?", importedModel.ID, channel.ID).First(&importedRoute).Error; err != nil || importedRoute.Status != "enabled" {
		t.Fatalf("imported route must be enabled by default: %#v, %v", importedRoute, err)
	}
	repeatedImport, errorData := svc.ImportChannelModels(ctx, channel.ID, dtos.ChannelModelImportInput{Models: testResult.Models})
	if errorData.IsNotNil() || repeatedImport.ModelsCreated != 0 || repeatedImport.RoutesCreated != 0 || repeatedImport.Skipped != 1 {
		t.Fatalf("repeated import must be idempotent: %#v, %v", repeatedImport, errorData.Err)
	}
	channels, errorData := svc.ListChannels(ctx, "", provider.ID, "")
	if errorData.IsNotNil() || len(channels.Data) != 1 || channels.Data[0].HealthStatus != "healthy" {
		t.Fatalf("list channels: %#v, %v", channels, errorData.Err)
	}
	if errorData = svc.DeleteProvider(ctx, provider.ID); errorData.IsNil() || errorData.ResponseCode != http.StatusConflict {
		t.Fatalf("referenced provider deletion must conflict: %#v", errorData)
	}
}

func TestSyncChannelModelsReconcilesMissingModelsWithoutDeletingSharedModels(t *testing.T) {
	db := dashboardTestDatabase(t)
	previousConfig := config.ApplicationConfig
	config.ApplicationConfig = &config.Config{Gateway: config.GatewayConfig{
		UpstreamTimeoutSeconds: 5, FailureThreshold: 3, CooldownSeconds: 60,
	}}
	t.Cleanup(func() { config.ApplicationConfig = previousConfig })

	var upstreamModels atomic.Value
	upstreamModels.Store([]string{"fresh", "shared", "gone"})
	var upstreamFails atomic.Bool
	var upstreamInvalid atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, _ *http.Request) {
		if upstreamFails.Load() {
			http.Error(resp, "unavailable", http.StatusServiceUnavailable)
			return
		}
		if upstreamInvalid.Load() {
			_, _ = resp.Write([]byte(`{}`))
			return
		}
		models := upstreamModels.Load().([]string)
		data := make([]map[string]string, 0, len(models))
		for _, name := range models {
			data = append(data, map[string]string{"id": name})
		}
		resp.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(resp).Encode(map[string]any{"object": "list", "data": data})
	}))
	defer server.Close()

	ctx := context.Background()
	svc := ControlPlaneService{}
	provider, errorData := svc.CreateProvider(ctx, dtos.ProviderInput{Name: "Sync Provider", Type: "openai-compatible", Status: "enabled", Config: map[string]any{}})
	if errorData.IsNotNil() {
		t.Fatalf("create provider: %v", errorData.Err)
	}
	primary, errorData := svc.CreateChannel(ctx, dtos.ChannelInput{ProviderID: provider.ID, Name: "primary", BaseURL: server.URL + "/v1", Priority: 10, Weight: 1, Status: "enabled"})
	if errorData.IsNotNil() {
		t.Fatalf("create primary channel: %v", errorData.Err)
	}

	first, errorData := svc.SyncChannelModels(ctx, primary.ID)
	if errorData.IsNotNil() || first.Discovered != 3 || first.ModelsCreated != 3 || first.RoutesCreated != 3 {
		t.Fatalf("initial sync: %#v, %v", first, errorData.Err)
	}
	var shared daos.AIModel
	if err := db.Where("name = ?", "shared").First(&shared).Error; err != nil || shared.Status != "active" {
		t.Fatalf("synchronized model must be active by default: %#v, %v", shared, err)
	}
	var primarySharedRoute daos.ModelRoute
	if err := db.Where("model_id = ? AND channel_id = ?", shared.ID, primary.ID).First(&primarySharedRoute).Error; err != nil || primarySharedRoute.Status != "enabled" {
		t.Fatalf("synchronized route must be enabled by default: %#v, %v", primarySharedRoute, err)
	}
	var fresh daos.AIModel
	if err := db.Where("name = ?", "fresh").First(&fresh).Error; err != nil {
		t.Fatalf("find fresh model: %v", err)
	}
	if err := db.Model(&fresh).Update("status", "disabled").Error; err != nil {
		t.Fatalf("disable existing fresh model: %v", err)
	}
	if err := db.Model(&daos.ModelRoute{}).
		Where("model_id = ? AND channel_id = ?", fresh.ID, primary.ID).
		Update("status", "disabled").Error; err != nil {
		t.Fatalf("disable existing fresh route: %v", err)
	}
	secondary, errorData := svc.CreateChannel(ctx, dtos.ChannelInput{ProviderID: provider.ID, Name: "secondary", BaseURL: server.URL + "/v1", Weight: 1, Status: "enabled"})
	if errorData.IsNotNil() {
		t.Fatalf("create secondary channel: %v", errorData.Err)
	}
	sharedRoute := daos.ModelRoute{
		GatewayRecord: daos.GatewayRecord{ID: "shared-secondary-route"},
		ModelID:       shared.ID, ChannelID: secondary.ID, UpstreamModel: "shared", Weight: 1, Status: "enabled",
	}
	if err := db.Create(&sharedRoute).Error; err != nil {
		t.Fatalf("create shared route: %v", err)
	}

	upstreamModels.Store([]string{"fresh"})
	second, errorData := svc.SyncChannelModels(ctx, primary.ID)
	if errorData.IsNotNil() || second.Discovered != 1 || second.RoutesDeleted != 2 || second.ModelsDeleted != 1 || second.Unchanged != 1 {
		t.Fatalf("reconcile sync: %#v, %v", second, errorData.Err)
	}
	var count int64
	if err := db.Model(&daos.AIModel{}).Where("name = ?", "gone").Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("orphaned missing model must be deleted: count=%d err=%v", count, err)
	}
	if err := db.Model(&daos.AIModel{}).Where("name = ?", "shared").Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("shared model must remain: count=%d err=%v", count, err)
	}
	if err := db.Where("id = ?", fresh.ID).First(&fresh).Error; err != nil || fresh.Status != "disabled" {
		t.Fatalf("sync must preserve an existing model's disabled status: %#v, %v", fresh, err)
	}
	var freshRoute daos.ModelRoute
	if err := db.Where("model_id = ? AND channel_id = ?", fresh.ID, primary.ID).First(&freshRoute).Error; err != nil || freshRoute.Status != "disabled" {
		t.Fatalf("sync must preserve an existing route's disabled status: %#v, %v", freshRoute, err)
	}
	primaryModels, errorData := svc.ListModels(ctx, "", "", primary.ID)
	if errorData.IsNotNil() || len(primaryModels.Data) != 1 || primaryModels.Data[0].Name != "fresh" || primaryModels.Data[0].ChannelName != "primary" {
		t.Fatalf("list primary models: %#v, %v", primaryModels, errorData.Err)
	}

	upstreamFails.Store(true)
	if _, failedSync := svc.SyncChannelModels(ctx, primary.ID); failedSync.IsNil() || failedSync.ResponseCode != http.StatusBadGateway {
		t.Fatalf("failed discovery must abort sync: %#v", failedSync)
	}
	if err := db.Model(&daos.ModelRoute{}).Where("channel_id = ?", primary.ID).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("failed discovery must not delete routes: count=%d err=%v", count, err)
	}

	upstreamFails.Store(false)
	upstreamInvalid.Store(true)
	if _, invalidSync := svc.SyncChannelModels(ctx, primary.ID); invalidSync.IsNil() || invalidSync.ResponseCode != http.StatusBadGateway {
		t.Fatalf("invalid discovery payload must abort sync: %#v", invalidSync)
	}
	if err := db.Model(&daos.ModelRoute{}).Where("channel_id = ?", primary.ID).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("invalid discovery payload must not delete routes: count=%d err=%v", count, err)
	}
}

func TestChannelCooldownRequiresHalfOpenProbe(t *testing.T) {
	db := dashboardTestDatabase(t)
	previousConfig := config.ApplicationConfig
	config.ApplicationConfig = &config.Config{Gateway: config.GatewayConfig{
		UpstreamTimeoutSeconds: 5, FailureThreshold: 2, CooldownSeconds: 60,
	}}
	t.Cleanup(func() { config.ApplicationConfig = previousConfig })

	var requests atomic.Int64
	var healthy atomic.Bool
	healthy.Store(true)
	server := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		if !healthy.Load() {
			http.Error(resp, "unavailable", http.StatusServiceUnavailable)
			return
		}
		resp.Header().Set("Content-Type", "application/json")
		_, _ = resp.Write([]byte(`{"object":"list","data":[]}`))
	}))
	defer server.Close()

	provider := daos.Provider{GatewayRecord: daos.GatewayRecord{ID: "half-provider"}, Name: "provider", Type: "openai-compatible", Status: "enabled", Config: "{}"}
	if err := db.Create(&provider).Error; err != nil {
		t.Fatalf("create provider: %v", err)
	}
	future := time.Now().Add(time.Minute)
	channel := daos.Channel{
		GatewayRecord: daos.GatewayRecord{ID: "half-channel"}, ProviderID: provider.ID, Name: "channel",
		BaseURL: server.URL + "/v1", Weight: 1, Status: "enabled", HealthStatus: "cooldown",
		CooldownUntil: &future, ConsecutiveFailures: 2,
	}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("create channel: %v", err)
	}

	result, errorData := (ControlPlaneService{}).TestChannel(context.Background(), channel.ID)
	if errorData.IsNotNil() || result.Success || requests.Load() != 0 {
		t.Fatalf("cooling channel must not be probed: %#v, requests=%d, err=%v", result, requests.Load(), errorData.Err)
	}

	expired := time.Now().Add(-time.Second)
	if err := db.Model(&daos.Channel{}).Where("id = ?", channel.ID).Update("cooldown_until", expired).Error; err != nil {
		t.Fatalf("expire cooldown: %v", err)
	}
	result, errorData = (ControlPlaneService{}).TestChannel(context.Background(), channel.ID)
	if errorData.IsNotNil() || !result.Success || requests.Load() != 1 {
		t.Fatalf("expired channel must receive one half-open probe: %#v, requests=%d, err=%v", result, requests.Load(), errorData.Err)
	}
	var observed daos.Channel
	if err := db.First(&observed, "id = ?", channel.ID).Error; err != nil || observed.HealthStatus != "healthy" || observed.CooldownUntil != nil || observed.ConsecutiveFailures != 0 {
		t.Fatalf("successful half-open probe must close the circuit: %#v, err=%v", observed, err)
	}

	healthy.Store(false)
	if err := db.Model(&daos.Channel{}).Where("id = ?", channel.ID).Updates(map[string]any{
		"health_status": "cooldown", "cooldown_until": expired, "consecutive_failures": 2,
	}).Error; err != nil {
		t.Fatalf("prepare failed half-open probe: %v", err)
	}
	result, errorData = (ControlPlaneService{}).TestChannel(context.Background(), channel.ID)
	if errorData.IsNotNil() || result.Success {
		t.Fatalf("expected failed half-open probe: %#v, err=%v", result, errorData.Err)
	}
	observed = daos.Channel{}
	if err := db.First(&observed, "id = ?", channel.ID).Error; err != nil || observed.HealthStatus != "cooldown" || observed.CooldownUntil == nil || !observed.CooldownUntil.After(time.Now()) {
		t.Fatalf("failed half-open probe must renew cooldown: %#v, err=%v", observed, err)
	}
}

func TestRunModelDiagnosticReturnsRouteTelemetry(t *testing.T) {
	db := dashboardTestDatabase(t)
	previousConfig := config.ApplicationConfig
	config.ApplicationConfig = &config.Config{Gateway: config.GatewayConfig{
		UpstreamTimeoutSeconds: 5, MaxAttempts: 2, MaxReservedTokens: 4096, FailureThreshold: 2, CooldownSeconds: 60,
	}}
	t.Cleanup(func() { config.ApplicationConfig = previousConfig })

	server := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/v1/chat/completions" {
			http.NotFound(resp, req)
			return
		}
		var payload map[string]any
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil || payload["model"] != "upstream-coder" || payload["stream"] != true {
			http.Error(resp, "invalid request", http.StatusBadRequest)
			return
		}
		resp.Header().Set("Content-Type", "text/event-stream")
		_, _ = resp.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\n"))
		_, _ = resp.Write([]byte("data: {\"choices\":[],\"usage\":{\"prompt_tokens\":4,\"completion_tokens\":2,\"total_tokens\":6}}\n\n"))
		_, _ = resp.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	provider := daos.Provider{GatewayRecord: daos.GatewayRecord{ID: "diag-provider"}, Name: "Provider", Type: "openai-compatible", Status: "enabled", Config: "{}"}
	channel := daos.Channel{GatewayRecord: daos.GatewayRecord{ID: "diag-channel"}, ProviderID: provider.ID, Name: "Channel", BaseURL: server.URL + "/v1", Weight: 1, Status: "enabled", HealthStatus: "healthy"}
	model := daos.AIModel{GatewayRecord: daos.GatewayRecord{ID: "diag-model"}, Name: "coder", DisplayName: "Coder", Modality: "chat", Capabilities: "[]", Status: "disabled"}
	route := daos.ModelRoute{GatewayRecord: daos.GatewayRecord{ID: "diag-route"}, ModelID: model.ID, ChannelID: channel.ID, UpstreamModel: "upstream-coder", Weight: 1, Status: "enabled"}
	for _, record := range []any{&provider, &channel, &model, &route} {
		if err := db.Create(record).Error; err != nil {
			t.Fatalf("create diagnostic fixture: %v", err)
		}
	}

	result, errorData := (ControlPlaneService{}).RunModelDiagnostic(context.Background(), dtos.ModelDiagnosticInput{
		ModelID: model.ID, Prompt: "say hello", MaxOutputTokens: 32,
	})
	if errorData.IsNotNil() || !result.Success {
		t.Fatalf("run diagnostic: %#v, err=%v", result, errorData.Err)
	}
	if result.OutputText != "hello" || result.TotalTokens != 6 || result.Estimated || result.FinalChannelID != channel.ID || len(result.Attempts) != 1 {
		t.Fatalf("unexpected diagnostic telemetry: %#v", result)
	}
	if result.Attempts[0].ProviderName != provider.Name || result.Attempts[0].UpstreamModel != route.UpstreamModel {
		t.Fatalf("missing internal route telemetry: %#v", result.Attempts[0])
	}
}

func TestControlPlaneListsAuditLogs(t *testing.T) {
	db := dashboardTestDatabase(t)
	account := daos.Account{ID: "admin-1", Username: "admin", Enable: true}
	if err := db.Create(&account).Error; err != nil {
		t.Fatalf("create audit operator: %v", err)
	}
	log := daos.AuditLog{
		GatewayRecord: daos.GatewayRecord{ID: "audit-1"}, OperatorID: account.ID,
		Action: "update", ResourceType: "channels", ResourceID: "channel-1", Method: "PUT",
		RoutePath: "/api/v1/channels/{id}", Summary: `{"route":"/api/v1/channels/{id}"}`,
		Result: "succeeded", StatusCode: 200, RequestID: "request-1", RemoteIP: "127.0.0.1",
	}
	if err := db.Create(&log).Error; err != nil {
		t.Fatalf("create audit log: %v", err)
	}
	result, errorData := (ControlPlaneService{}).ListAuditLogs(context.Background(), 1, 20, account.ID, "update", "channels", "succeeded")
	if errorData.IsNotNil() || result.Total != 1 || result.Data[0].OperatorUsername != "admin" || result.Data[0].Summary["route"] == "" {
		t.Fatalf("list audit logs: %#v, %v", result, errorData.Err)
	}
}
