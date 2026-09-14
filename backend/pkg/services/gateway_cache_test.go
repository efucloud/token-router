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
		Enabled: true, Address: server.Addr(), KeyPrefix: "gateway-cache-test", RouteTTLSeconds: 60,
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
