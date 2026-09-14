package services

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/efucloud/token-router/pkg/config"
	"github.com/efucloud/token-router/pkg/models/daos"
	"github.com/redis/go-redis/v9"
)

func setupGatewayRedisCache(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	server := miniredis.RunT(t)
	previousClient := config.RedisClient
	previousConfig := config.ApplicationConfig
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	config.RedisClient = client
	config.ApplicationConfig = &config.Config{Redis: &config.RedisConfig{
		Enabled: true, Addresses: []string{server.Addr()},
	}}
	t.Cleanup(func() {
		_ = client.Close()
		config.RedisClient = previousClient
		config.ApplicationConfig = previousConfig
	})
	return server
}

func seedGatewayCacheRoute(t *testing.T) GatewayPrincipal {
	t.Helper()
	db := dashboardTestDatabase(t)
	provider := daos.Provider{GatewayRecord: daos.GatewayRecord{ID: "provider-cache"}, Name: "provider", Type: "openai-compatible", Status: "enabled", Config: "{}"}
	channel := daos.Channel{
		GatewayRecord: daos.GatewayRecord{ID: "channel-cache"}, ProviderID: provider.ID, Name: "channel",
		BaseURL: "https://old.example.com/v1", Status: "enabled", HealthStatus: "healthy", Weight: 1,
	}
	model := daos.AIModel{GatewayRecord: daos.GatewayRecord{ID: "model-cache"}, Name: "cached-model", DisplayName: "Cached", Modality: "chat", Capabilities: "[]", Status: "active"}
	route := daos.ModelRoute{
		GatewayRecord: daos.GatewayRecord{ID: "route-cache"}, ModelID: model.ID, ChannelID: channel.ID,
		UpstreamModel: "upstream-cached", Weight: 1, Status: "enabled",
	}
	for _, value := range []any{&provider, &channel, &model, &route} {
		if err := db.Create(value).Error; err != nil {
			t.Fatalf("seed gateway cache route: %v", err)
		}
	}
	return GatewayPrincipal{SystemToken: true}
}

func TestResolveModelAndRoutesUsesRedisAndInvalidatesByGeneration(t *testing.T) {
	server := setupGatewayRedisCache(t)
	principal := seedGatewayCacheRoute(t)
	ctx := context.Background()
	service := GatewayService{}

	_, routes, err := service.ResolveModelAndRoutes(ctx, principal, "cached-model", "chat")
	if err != nil || len(routes) != 1 || routes[0].BaseURL != "https://old.example.com/v1" {
		t.Fatalf("prime route cache: %#v, %v", routes, err)
	}
	if len(server.Keys()) < 2 {
		t.Fatalf("expected version and route keys, got %v", server.Keys())
	}
	if err = config.DBConnect.Model(&daos.Channel{}).Where("id = ?", "channel-cache").Update("base_url", "https://new.example.com/v1").Error; err != nil {
		t.Fatalf("update database route: %v", err)
	}
	_, routes, err = service.ResolveModelAndRoutes(ctx, principal, "cached-model", "chat")
	if err != nil || routes[0].BaseURL != "https://old.example.com/v1" {
		t.Fatalf("expected cached route before invalidation: %#v, %v", routes, err)
	}

	InvalidateGatewayRouteCache(ctx)
	_, routes, err = service.ResolveModelAndRoutes(ctx, principal, "cached-model", "chat")
	if err != nil || routes[0].BaseURL != "https://new.example.com/v1" {
		t.Fatalf("expected refreshed route after invalidation: %#v, %v", routes, err)
	}
}

func TestResolveModelAndRoutesFallsBackWhenRedisIsUnavailable(t *testing.T) {
	server := setupGatewayRedisCache(t)
	principal := seedGatewayCacheRoute(t)
	server.Close()

	_, routes, err := (GatewayService{}).ResolveModelAndRoutes(context.Background(), principal, "cached-model", "chat")
	if err != nil || len(routes) != 1 || routes[0].ChannelID != "channel-cache" {
		t.Fatalf("database fallback failed: %#v, %v", routes, err)
	}
}

func TestPublishedModelsCacheKeepsPrincipalFiltering(t *testing.T) {
	setupGatewayRedisCache(t)
	seedGatewayCacheRoute(t)
	service := GatewayService{}
	ctx := context.Background()

	models, err := service.PublishedModels(ctx, GatewayPrincipal{SystemToken: true})
	if err != nil || len(models) != 1 {
		t.Fatalf("prime published model cache: %#v, %v", models, err)
	}
	denied := GatewayPrincipal{Token: daos.APIToken{AllowedModels: `["another-model"]`}}
	models, err = service.PublishedModels(ctx, denied)
	if err != nil || len(models) != 0 {
		t.Fatalf("cached models leaked across principals: %#v, %v", models, err)
	}
}

func TestPublishedModelsExcludeDisabledResources(t *testing.T) {
	db := dashboardTestDatabase(t)
	providers := []daos.Provider{
		{GatewayRecord: daos.GatewayRecord{ID: "provider-enabled"}, Name: "enabled", Type: "openai-compatible", Status: "enabled", Config: "{}"},
		{GatewayRecord: daos.GatewayRecord{ID: "provider-disabled"}, Name: "disabled", Type: "openai-compatible", Status: "disabled", Config: "{}"},
	}
	channels := []daos.Channel{
		{GatewayRecord: daos.GatewayRecord{ID: "channel-enabled"}, ProviderID: "provider-enabled", Name: "enabled", BaseURL: "https://enabled.example.com/v1", Status: "enabled", HealthStatus: "healthy", Weight: 1},
		{GatewayRecord: daos.GatewayRecord{ID: "channel-disabled"}, ProviderID: "provider-enabled", Name: "disabled", BaseURL: "https://disabled.example.com/v1", Status: "disabled", HealthStatus: "healthy", Weight: 1},
		{GatewayRecord: daos.GatewayRecord{ID: "channel-provider-disabled"}, ProviderID: "provider-disabled", Name: "provider-disabled", BaseURL: "https://provider-disabled.example.com/v1", Status: "enabled", HealthStatus: "healthy", Weight: 1},
		{GatewayRecord: daos.GatewayRecord{ID: "channel-cooldown"}, ProviderID: "provider-enabled", Name: "cooldown", BaseURL: "https://cooldown.example.com/v1", Status: "enabled", HealthStatus: "cooldown", Weight: 1},
	}
	models := []daos.AIModel{
		{GatewayRecord: daos.GatewayRecord{ID: "model-available"}, Name: "available", DisplayName: "Available", Modality: "chat", Capabilities: "[]", Status: "active"},
		{GatewayRecord: daos.GatewayRecord{ID: "model-disabled"}, Name: "model-disabled", DisplayName: "Disabled model", Modality: "chat", Capabilities: "[]", Status: "disabled"},
		{GatewayRecord: daos.GatewayRecord{ID: "model-route-disabled"}, Name: "route-disabled", DisplayName: "Disabled route", Modality: "chat", Capabilities: "[]", Status: "active"},
		{GatewayRecord: daos.GatewayRecord{ID: "model-channel-disabled"}, Name: "channel-disabled", DisplayName: "Disabled channel", Modality: "chat", Capabilities: "[]", Status: "active"},
		{GatewayRecord: daos.GatewayRecord{ID: "model-provider-disabled"}, Name: "provider-disabled", DisplayName: "Disabled provider", Modality: "chat", Capabilities: "[]", Status: "active"},
		{GatewayRecord: daos.GatewayRecord{ID: "model-cooldown"}, Name: "cooldown", DisplayName: "Cooldown", Modality: "chat", Capabilities: "[]", Status: "active"},
	}
	routes := []daos.ModelRoute{
		{GatewayRecord: daos.GatewayRecord{ID: "route-available"}, ModelID: "model-available", ChannelID: "channel-enabled", UpstreamModel: "available", Weight: 1, Status: "enabled"},
		{GatewayRecord: daos.GatewayRecord{ID: "route-model-disabled"}, ModelID: "model-disabled", ChannelID: "channel-enabled", UpstreamModel: "model-disabled", Weight: 1, Status: "enabled"},
		{GatewayRecord: daos.GatewayRecord{ID: "route-disabled"}, ModelID: "model-route-disabled", ChannelID: "channel-enabled", UpstreamModel: "route-disabled", Weight: 1, Status: "disabled"},
		{GatewayRecord: daos.GatewayRecord{ID: "route-channel-disabled"}, ModelID: "model-channel-disabled", ChannelID: "channel-disabled", UpstreamModel: "channel-disabled", Weight: 1, Status: "enabled"},
		{GatewayRecord: daos.GatewayRecord{ID: "route-provider-disabled"}, ModelID: "model-provider-disabled", ChannelID: "channel-provider-disabled", UpstreamModel: "provider-disabled", Weight: 1, Status: "enabled"},
		{GatewayRecord: daos.GatewayRecord{ID: "route-cooldown"}, ModelID: "model-cooldown", ChannelID: "channel-cooldown", UpstreamModel: "cooldown", Weight: 1, Status: "enabled"},
	}
	for _, value := range []any{&providers, &channels, &models, &routes} {
		if err := db.Create(value).Error; err != nil {
			t.Fatalf("seed published model visibility: %v", err)
		}
	}

	published, err := (GatewayService{}).PublishedModels(context.Background(), GatewayPrincipal{SystemToken: true})
	if err != nil {
		t.Fatalf("list published models: %v", err)
	}
	if len(published) != 1 || published[0].Name != "available" {
		t.Fatalf("disabled resources leaked into published models: %#v", published)
	}
}
