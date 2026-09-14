package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/efucloud/token-router/pkg/config"
	"github.com/redis/go-redis/v9"
)

const gatewayRouteCacheVersionKey = "gateway-routes:version"

const (
	gatewayRouteCachePrefix = "token-router"
	gatewayRouteCacheTTL    = 30 * time.Second
)

type gatewayRouteCacheEntry struct {
	Model  GatewayModel   `json:"model"`
	Routes []GatewayRoute `json:"routes"`
}

func gatewayCacheKey(parts ...string) string {
	return gatewayRouteCachePrefix + ":" + strings.Join(parts, ":")
}

func gatewayRouteCacheVersion(ctx context.Context) (string, bool) {
	if config.RedisClient == nil {
		return "", false
	}
	key := gatewayCacheKey(gatewayRouteCacheVersionKey)
	version, err := config.RedisClient.Get(ctx, key).Result()
	if err == nil {
		return version, true
	}
	if err != redis.Nil {
		return "", false
	}
	if err = config.RedisClient.SetNX(ctx, key, "1", 0).Err(); err != nil {
		return "", false
	}
	version, err = config.RedisClient.Get(ctx, key).Result()
	return version, err == nil
}

func gatewayRouteCacheHash(values ...string) string {
	digest := sha256.Sum256([]byte(strings.Join(values, "\x00")))
	return hex.EncodeToString(digest[:])
}

func readGatewayRouteCache(ctx context.Context, name, modality string) (GatewayModel, []GatewayRoute, bool) {
	version, enabled := gatewayRouteCacheVersion(ctx)
	if !enabled {
		return GatewayModel{}, nil, false
	}
	key := gatewayCacheKey("gateway-routes", version, gatewayRouteCacheHash(name, modality))
	data, err := config.RedisClient.Get(ctx, key).Bytes()
	if err != nil {
		return GatewayModel{}, nil, false
	}
	var entry gatewayRouteCacheEntry
	if json.Unmarshal(data, &entry) != nil || entry.Model.Name == "" || len(entry.Routes) == 0 {
		_ = config.RedisClient.Del(ctx, key).Err()
		return GatewayModel{}, nil, false
	}
	return entry.Model, entry.Routes, true
}

func writeGatewayRouteCache(ctx context.Context, version, name, modality string, model GatewayModel, routes []GatewayRoute) {
	if config.RedisClient == nil || version == "" || len(routes) == 0 {
		return
	}
	data, err := json.Marshal(gatewayRouteCacheEntry{Model: model, Routes: routes})
	if err != nil {
		return
	}
	key := gatewayCacheKey("gateway-routes", version, gatewayRouteCacheHash(name, modality))
	_ = config.RedisClient.Set(ctx, key, data, gatewayRouteCacheTTL).Err()
}

func readPublishedModelsCache(ctx context.Context) ([]GatewayModel, bool) {
	version, enabled := gatewayRouteCacheVersion(ctx)
	if !enabled {
		return nil, false
	}
	data, err := config.RedisClient.Get(ctx, gatewayCacheKey("gateway-models", version)).Bytes()
	if err != nil {
		return nil, false
	}
	var models []GatewayModel
	if json.Unmarshal(data, &models) != nil {
		return nil, false
	}
	return models, true
}

func writePublishedModelsCache(ctx context.Context, version string, models []GatewayModel) {
	if config.RedisClient == nil || version == "" {
		return
	}
	data, err := json.Marshal(models)
	if err != nil {
		return
	}
	_ = config.RedisClient.Set(ctx, gatewayCacheKey("gateway-models", version), data, gatewayRouteCacheTTL).Err()
}

// InvalidateGatewayRouteCache advances a shared generation instead of scanning
// keys. Old entries expire naturally and all instances immediately use the new
// generation.
func InvalidateGatewayRouteCache(ctx context.Context) {
	if config.RedisClient == nil {
		return
	}
	key := gatewayCacheKey(gatewayRouteCacheVersionKey)
	if _, err := config.RedisClient.Incr(ctx, key).Result(); err == nil {
		return
	}
	// A best-effort invalidation must never make control-plane writes fail.
	_ = config.RedisClient.Set(ctx, key, strconv.FormatInt(time.Now().UnixNano(), 10), 0).Err()
}
