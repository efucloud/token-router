package services

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/efucloud/token-router/pkg/config"
	"github.com/efucloud/token-router/pkg/models/daos"
	"github.com/efucloud/token-router/pkg/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GatewayService struct{}

type GatewayPrincipal struct {
	Token       daos.APIToken
	Account     daos.Account
	SystemToken bool
}

const systemTokenUsageID = "system"

const GatewayConversationHeader = "X-Token-Router-Conversation-Id"

var gatewayChatToolNameInvalid = regexp.MustCompile(`[^A-Za-z0-9_-]`)

type GatewayModel struct {
	ID              string
	Name            string
	DisplayName     string
	Description     string
	Modality        string
	ContextWindow   int64
	MaxOutputTokens int64
	Capabilities    string
}

type GatewayRoute struct {
	RouteID             string
	ChannelID           string
	UpstreamModel       string
	Priority            int
	Weight              int
	BaseURL             string
	EncryptedAPIKey     string
	TimeoutSeconds      int
	HealthStatus        string
	ConsecutiveFailures int
}

type GatewayUsage struct {
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
	TotalTokens      int64 `json:"total_tokens"`
}

type GatewayReservation struct {
	RequestID          string
	UsageLogID         string
	AccountID          string
	APITokenID         string
	ReservedTokens     int64
	StartedAt          time.Time
	RateWindow         time.Time
	RateLimited        bool
	ConcurrencyLimited bool
}

type GatewayQuotaError struct {
	Dimension string
	Limit     int64
	ResetAt   time.Time
}

func (GatewayService) AuthorizeIP(principal GatewayPrincipal, remoteAddress string) error {
	if principal.SystemToken {
		return nil
	}
	if strings.TrimSpace(principal.Token.IPAllowlist) == "" || strings.TrimSpace(principal.Token.IPAllowlist) == "[]" {
		return nil
	}
	host := remoteAddress
	if parsedHost, _, err := net.SplitHostPort(remoteAddress); err == nil {
		host = parsedHost
	}
	remoteIP := net.ParseIP(strings.Trim(host, "[]"))
	if remoteIP == nil {
		return errors.New("request IP is unavailable")
	}
	var allowlist []string
	if json.Unmarshal([]byte(principal.Token.IPAllowlist), &allowlist) != nil {
		return errors.New("API key IP allowlist is invalid")
	}
	for _, value := range allowlist {
		if allowedIP := net.ParseIP(value); allowedIP != nil && allowedIP.Equal(remoteIP) {
			return nil
		}
		if _, network, err := net.ParseCIDR(value); err == nil && network.Contains(remoteIP) {
			return nil
		}
	}
	return errors.New("request IP is not allowed by this API key")
}

func (e GatewayQuotaError) Error() string {
	return fmt.Sprintf("%s quota exceeded", e.Dimension)
}

func (GatewayService) Authenticate(ctx context.Context, plaintext string) (GatewayPrincipal, error) {
	hash, err := utils.HashAPIToken(plaintext)
	if err != nil {
		return GatewayPrincipal{}, err
	}
	var token daos.APIToken
	if err = config.DBConnect.WithContext(ctx).Where("key_hash = ?", hash).First(&token).Error; err != nil {
		return GatewayPrincipal{}, errors.New("invalid API key")
	}
	if token.Status != "active" || (token.ExpiresAt != nil && !token.ExpiresAt.After(time.Now())) {
		return GatewayPrincipal{}, errors.New("API key is disabled or expired")
	}
	var account daos.Account
	if err = config.DBConnect.WithContext(ctx).Where("id = ?", token.AccountID).First(&account).Error; err != nil {
		return GatewayPrincipal{}, errors.New("API key account does not exist")
	}
	if !account.Enable {
		return GatewayPrincipal{}, errors.New("API key account is disabled")
	}
	return GatewayPrincipal{Token: token, Account: account}, nil
}

func (GatewayService) AuthenticateSystemToken(ctx context.Context, accountID string) (GatewayPrincipal, error) {
	if strings.TrimSpace(accountID) == "" {
		return GatewayPrincipal{}, errors.New("system token account is missing")
	}
	var account daos.Account
	if err := config.DBConnect.WithContext(ctx).Where("id = ?", accountID).First(&account).Error; err != nil {
		return GatewayPrincipal{}, errors.New("system token account does not exist")
	}
	if !account.Enable {
		return GatewayPrincipal{}, errors.New("system token account is disabled")
	}
	return GatewayPrincipal{Account: account, SystemToken: true}, nil
}

func tokenAllowsModel(token daos.APIToken, model string) bool {
	if strings.TrimSpace(token.AllowedModels) == "" || strings.TrimSpace(token.AllowedModels) == "[]" {
		return true
	}
	var allowed []string
	if json.Unmarshal([]byte(token.AllowedModels), &allowed) != nil {
		return false
	}
	for _, item := range allowed {
		if item == model {
			return true
		}
	}
	return false
}

func (GatewayService) PublishedModels(ctx context.Context, principal GatewayPrincipal) ([]GatewayModel, error) {
	var models []GatewayModel
	if cached, ok := readPublishedModelsCache(ctx); ok {
		models = cached
	} else {
		cacheVersion, _ := gatewayRouteCacheVersion(ctx)
		err := config.DBConnect.WithContext(ctx).Table("ai_model AS m").Distinct().
			Select("m.id, m.name, m.display_name, m.description, m.modality, m.context_window, m.max_output_tokens, m.capabilities").
			Joins("JOIN model_route AS r ON r.model_id = m.id AND r.status = ?", "enabled").
			Joins("JOIN channel AS c ON c.id = r.channel_id AND c.status = ?", "enabled").
			Joins("JOIN provider AS p ON p.id = c.provider_id AND p.status = ?", "enabled").
			Where("m.status = ?", "active").
			Where("c.health_status NOT IN ?", []string{"cooldown", "half_open"}).
			Order("m.name ASC").Find(&models).Error
		if err != nil {
			return nil, err
		}
		writePublishedModelsCache(ctx, cacheVersion, models)
	}
	result := make([]GatewayModel, 0, len(models))
	for _, model := range models {
		if principal.SystemToken || tokenAllowsModel(principal.Token, model.Name) {
			result = append(result, model)
		}
	}
	return result, nil
}

func (GatewayService) ResolveModelAndRoutes(ctx context.Context, principal GatewayPrincipal, name, modality string) (GatewayModel, []GatewayRoute, error) {
	if !principal.SystemToken && !tokenAllowsModel(principal.Token, name) {
		return GatewayModel{}, nil, errors.New("model is not allowed by this API key")
	}
	if model, routes, ok := readGatewayRouteCache(ctx, name, modality); ok {
		return model, routes, nil
	}
	cacheVersion, _ := gatewayRouteCacheVersion(ctx)
	var model GatewayModel
	err := config.DBConnect.WithContext(ctx).Table("ai_model").
		Select("id, name, display_name, modality, capabilities").
		Where("name = ? AND status = ?", name, "active").First(&model).Error
	if err != nil {
		return GatewayModel{}, nil, errors.New("model is not published")
	}
	if model.Modality != modality {
		return GatewayModel{}, nil, fmt.Errorf("model %s does not support %s", name, modality)
	}
	var routes []GatewayRoute
	err = config.DBConnect.WithContext(ctx).Table("model_route AS r").
		Select(`r.id AS route_id, r.channel_id, r.upstream_model,
            CASE WHEN r.priority = 0 THEN c.priority ELSE r.priority END AS priority,
			r.weight, c.base_url, c.encrypted_api_key, c.timeout_seconds, c.health_status, c.consecutive_failures`).
		Joins("JOIN channel AS c ON c.id = r.channel_id").
		Joins("JOIN provider AS p ON p.id = c.provider_id").
		Where("r.model_id = ? AND r.status = ?", model.ID, "enabled").
		Where("c.status = ? AND p.status = ?", "enabled", "enabled").
		Where("c.health_status NOT IN ?", []string{"cooldown", "half_open"}).
		Find(&routes).Error
	if err != nil {
		return GatewayModel{}, nil, err
	}
	if len(routes) == 0 {
		return GatewayModel{}, nil, errors.New("no route is available for this model")
	}
	writeGatewayRouteCache(ctx, cacheVersion, name, modality, model, routes)
	return model, routes, nil
}

func estimateGatewayTokens(body []byte, maxReserved int64) int64 {
	var request struct {
		MaxTokens           int64 `json:"max_tokens"`
		MaxCompletionTokens int64 `json:"max_completion_tokens"`
		MaxOutputTokens     int64 `json:"max_output_tokens"`
	}
	_ = json.Unmarshal(body, &request)
	output := request.MaxCompletionTokens
	if output <= 0 {
		output = request.MaxTokens
	}
	if output <= 0 {
		output = request.MaxOutputTokens
	}
	if output <= 0 {
		output = 1024
	}
	estimate := int64(len(body))/3 + output + 64
	if maxReserved > 0 && estimate > maxReserved {
		estimate = maxReserved
	}
	if estimate < 1 {
		return 1
	}
	return estimate
}

func quotaAvailable(limit, used, reserved, addition int64) bool {
	return limit <= 0 || used+reserved+addition <= limit
}

func reserveRateAndConcurrency(tx *gorm.DB, token daos.APIToken, reservation *GatewayReservation, estimate int64) error {
	now := time.Now().UTC()
	if token.RPM > 0 || token.TPM > 0 {
		window := now.Truncate(time.Minute)
		seed := daos.APIRateLimitBucket{
			GatewayRecord: daos.GatewayRecord{ID: utils.GenerateDatabaseId()}, APITokenID: token.ID, WindowStart: window,
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&seed).Error; err != nil {
			return err
		}
		var bucket daos.APIRateLimitBucket
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("api_token_id = ? AND window_start = ?", token.ID, window).First(&bucket).Error; err != nil {
			return err
		}
		resetAt := window.Add(time.Minute)
		if token.RPM > 0 && bucket.UsedRequests+1 > int64(token.RPM) {
			return GatewayQuotaError{Dimension: "API key RPM", Limit: int64(token.RPM), ResetAt: resetAt}
		}
		if token.TPM > 0 && bucket.UsedTokens+bucket.ReservedTokens+estimate > token.TPM {
			return GatewayQuotaError{Dimension: "API key TPM", Limit: token.TPM, ResetAt: resetAt}
		}
		if err := tx.Model(&daos.APIRateLimitBucket{}).Where("id = ?", bucket.ID).Updates(map[string]any{
			"used_requests": bucket.UsedRequests + 1, "reserved_tokens": bucket.ReservedTokens + estimate,
		}).Error; err != nil {
			return err
		}
		reservation.RateWindow = window
		reservation.RateLimited = true
	}
	if token.MaxConcurrency > 0 {
		if err := tx.Where("api_token_id = ? AND expires_at <= ?", token.ID, now).Delete(&daos.ConcurrencyLease{}).Error; err != nil {
			return err
		}
		var active int64
		if err := tx.Model(&daos.ConcurrencyLease{}).Where("api_token_id = ? AND expires_at > ?", token.ID, now).Count(&active).Error; err != nil {
			return err
		}
		if active >= int64(token.MaxConcurrency) {
			return GatewayQuotaError{Dimension: "API key concurrency", Limit: int64(token.MaxConcurrency), ResetAt: now.Add(time.Second)}
		}
		leaseSeconds := config.ApplicationConfig.Gateway.UpstreamTimeoutSeconds*config.ApplicationConfig.Gateway.MaxAttempts + 60
		if leaseSeconds < 600 {
			leaseSeconds = 600
		}
		if leaseSeconds > 3600 {
			leaseSeconds = 3600
		}
		lease := daos.ConcurrencyLease{GatewayRecord: daos.GatewayRecord{ID: reservation.RequestID}, APITokenID: token.ID, ExpiresAt: now.Add(time.Duration(leaseSeconds) * time.Second)}
		if err := tx.Create(&lease).Error; err != nil {
			return err
		}
		reservation.ConcurrencyLimited = true
	}
	return nil
}

func (GatewayService) Reserve(ctx context.Context, principal GatewayPrincipal, model GatewayModel, endpoint string, streaming bool, estimate int64) (GatewayReservation, error) {
	reservation := GatewayReservation{
		RequestID:      utils.GenerateDatabaseId(),
		UsageLogID:     utils.GenerateDatabaseId(),
		AccountID:      principal.Account.ID,
		APITokenID:     principal.Token.ID,
		ReservedTokens: estimate,
		StartedAt:      time.Now(),
	}
	err := config.DBConnect.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var account daos.Account
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", reservation.AccountID).First(&account).Error; err != nil {
			return err
		}
		if !quotaAvailable(account.TokenLimit, account.UsedTokens, account.ReservedTokens, estimate) {
			return GatewayQuotaError{Dimension: "user token"}
		}
		if !quotaAvailable(account.RequestLimit, account.UsedRequests, account.ReservedRequests, 1) {
			return GatewayQuotaError{Dimension: "user request"}
		}
		var token daos.APIToken
		if !principal.SystemToken {
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", reservation.APITokenID).First(&token).Error; err != nil {
				return err
			}
			if !quotaAvailable(token.TokenLimit, token.UsedTokens, token.ReservedTokens, estimate) {
				return GatewayQuotaError{Dimension: "API key"}
			}
			if !quotaAvailable(token.RequestLimit, token.UsedRequests, token.ReservedRequests, 1) {
				return GatewayQuotaError{Dimension: "API key request"}
			}
			if err := reserveRateAndConcurrency(tx, token, &reservation, estimate); err != nil {
				return err
			}
		}
		if err := tx.Model(&daos.Account{}).Where("id = ?", account.ID).Updates(map[string]any{
			"reserved_tokens":   account.ReservedTokens + estimate,
			"reserved_requests": account.ReservedRequests + 1,
		}).Error; err != nil {
			return err
		}
		if !principal.SystemToken {
			if err := tx.Model(&daos.APIToken{}).Where("id = ?", token.ID).Updates(map[string]any{
				"reserved_tokens":   token.ReservedTokens + estimate,
				"reserved_requests": token.ReservedRequests + 1,
			}).Error; err != nil {
				return err
			}
		}
		usageTokenID, usageTokenPrefix := token.ID, token.KeyPrefix
		if principal.SystemToken {
			usageTokenID, usageTokenPrefix = systemTokenUsageID, systemTokenUsageID
		}
		usage := daos.UsageLog{
			GatewayRecord: daos.GatewayRecord{ID: reservation.UsageLogID},
			RequestID:     reservation.RequestID, AccountID: account.ID, APITokenID: usageTokenID,
			TokenPrefix: usageTokenPrefix, ModelID: model.ID, ModelName: model.Name,
			Endpoint: endpoint, Streaming: streaming, ReservedTokens: estimate, Status: "reserved",
		}
		return tx.Create(&usage).Error
	})
	return reservation, err
}

func nonNegative(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}

func (GatewayService) Settle(ctx context.Context, reservation GatewayReservation, usage GatewayUsage, success, estimated bool, httpStatus int, errorCode, errorSummary, channelID string) error {
	return config.DBConnect.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var log daos.UsageLog
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("request_id = ?", reservation.RequestID).First(&log).Error; err != nil {
			return err
		}
		if log.Status != "reserved" {
			return nil
		}
		var account daos.Account
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", reservation.AccountID).First(&account).Error; err != nil {
			return err
		}
		accountUpdates := map[string]any{
			"reserved_tokens":   nonNegative(account.ReservedTokens - reservation.ReservedTokens),
			"reserved_requests": nonNegative(account.ReservedRequests - 1),
		}
		status := "failed"
		if success {
			status = "succeeded"
			accountUpdates["used_tokens"] = account.UsedTokens + usage.TotalTokens
			accountUpdates["used_requests"] = account.UsedRequests + 1
		}
		if err := tx.Model(&daos.Account{}).Where("id = ?", account.ID).Updates(accountUpdates).Error; err != nil {
			return err
		}
		if reservation.APITokenID != "" {
			var token daos.APIToken
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", reservation.APITokenID).First(&token).Error; err != nil {
				return err
			}
			tokenUpdates := map[string]any{
				"reserved_tokens":   nonNegative(token.ReservedTokens - reservation.ReservedTokens),
				"reserved_requests": nonNegative(token.ReservedRequests - 1),
			}
			if success {
				tokenUpdates["used_tokens"] = token.UsedTokens + usage.TotalTokens
				tokenUpdates["used_requests"] = token.UsedRequests + 1
			}
			if err := tx.Model(&daos.APIToken{}).Where("id = ?", token.ID).Updates(tokenUpdates).Error; err != nil {
				return err
			}
		}
		if reservation.RateLimited {
			var bucket daos.APIRateLimitBucket
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("api_token_id = ? AND window_start = ?", reservation.APITokenID, reservation.RateWindow).First(&bucket).Error; err != nil {
				return err
			}
			updates := map[string]any{"reserved_tokens": nonNegative(bucket.ReservedTokens - reservation.ReservedTokens)}
			if success {
				updates["used_tokens"] = bucket.UsedTokens + usage.TotalTokens
			}
			if err := tx.Model(&daos.APIRateLimitBucket{}).Where("id = ?", bucket.ID).Updates(updates).Error; err != nil {
				return err
			}
		}
		if reservation.ConcurrencyLimited {
			if err := tx.Where("id = ? AND api_token_id = ?", reservation.RequestID, reservation.APITokenID).Delete(&daos.ConcurrencyLease{}).Error; err != nil {
				return err
			}
		}
		return tx.Model(&daos.UsageLog{}).Where("id = ?", log.ID).Updates(map[string]any{
			"channel_id": channelID, "prompt_tokens": usage.PromptTokens,
			"completion_tokens": usage.CompletionTokens, "total_tokens": usage.TotalTokens,
			"estimated": estimated, "status": status, "http_status": httpStatus,
			"duration_ms": time.Since(reservation.StartedAt).Milliseconds(), "error_code": errorCode,
			"error_summary": errorSummary,
		}).Error
	})
}

func selectGatewayRoute(routes []GatewayRoute) (GatewayRoute, []GatewayRoute, error) {
	if len(routes) == 0 {
		return GatewayRoute{}, nil, errors.New("no route remains")
	}
	maxPriority := routes[0].Priority
	for _, route := range routes[1:] {
		if route.Priority > maxPriority {
			maxPriority = route.Priority
		}
	}
	var group []GatewayRoute
	var rest []GatewayRoute
	weightTotal := 0
	for _, route := range routes {
		if route.Priority == maxPriority {
			if route.Weight <= 0 {
				route.Weight = 1
			}
			group = append(group, route)
			weightTotal += route.Weight
		} else {
			rest = append(rest, route)
		}
	}
	random, err := rand.Int(rand.Reader, big.NewInt(int64(weightTotal)))
	if err != nil {
		return GatewayRoute{}, nil, err
	}
	position := int(random.Int64())
	selectedIndex := 0
	for index, route := range group {
		if position < route.Weight {
			selectedIndex = index
			break
		}
		position -= route.Weight
	}
	selected := group[selectedIndex]
	rest = append(rest, group[:selectedIndex]...)
	rest = append(rest, group[selectedIndex+1:]...)
	return selected, rest, nil
}

func gatewayUpstreamURL(baseURL, endpoint string) (string, error) {
	parsed, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil {
		return "", errors.New("invalid upstream base URL")
	}
	basePath := strings.TrimRight(parsed.Path, "/")
	if !strings.HasSuffix(basePath, "/v1") {
		basePath += "/v1"
	}
	parsed.Path = basePath + endpoint
	return parsed.String(), nil
}

func rewriteGatewayRequest(body []byte, upstreamModel, endpoint string, streaming bool) ([]byte, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	payload["model"] = upstreamModel
	if streaming && endpoint == "/chat/completions" {
		options, _ := payload["stream_options"].(map[string]any)
		if options == nil {
			options = make(map[string]any)
		}
		options["include_usage"] = true
		payload["stream_options"] = options
	}
	return json.Marshal(payload)
}

func gatewayChatToolName(server, tool string) string {
	name := gatewayChatToolNameInvalid.ReplaceAllString("mcp__"+server+"__"+tool, "_")
	if len(name) > 64 {
		return name[:64]
	}
	return name
}

// InjectConversationTools keeps tool definitions under server control. The
// browser identifies the persisted conversation and only controls tool use
// through tool_choice.
func (GatewayService) InjectConversationTools(ctx context.Context, principal GatewayPrincipal, conversationID string, body []byte) ([]byte, error) {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return body, nil
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	delete(payload, "tools")
	if _, requested := payload["tool_choice"]; !requested {
		return json.Marshal(payload)
	}

	var conversation daos.ChatConversation
	if err := config.DBConnect.WithContext(ctx).
		Where("id = ? AND account_id = ?", conversationID, principal.Account.ID).
		First(&conversation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("chat conversation does not exist")
		}
		return nil, err
	}
	selected := chatConversationMCPServers(conversation, defaultChatMCPServers(ctx))
	selectedSet := make(map[string]struct{}, len(selected))
	for _, server := range selected {
		selectedSet[server] = struct{}{}
	}

	chatCtx := context.WithValue(ctx, config.RequestUserId, principal.Account.ID)
	capabilities, err := (ChatService{}).Capabilities(chatCtx)
	if err != nil {
		return nil, err
	}
	tools := make([]map[string]any, 0)
	for _, server := range capabilities.MCPServers {
		if server.Status != "connected" {
			continue
		}
		if _, enabled := selectedSet[server.Name]; !enabled {
			continue
		}
		for _, tool := range server.Tools {
			tools = append(tools, map[string]any{
				"type": "function",
				"function": map[string]any{
					"name": gatewayChatToolName(server.Name, tool.Name), "description": tool.Description,
					"parameters": tool.InputSchema,
				},
			})
		}
	}
	if len(tools) > 0 {
		payload["tools"] = tools
	}
	return json.Marshal(payload)
}

func createRouteAttempt(ctx context.Context, requestID string, route GatewayRoute, sequence, status int, duration time.Duration, errorCode, summary string) {
	_ = config.DBConnect.WithContext(ctx).Create(&daos.RouteAttempt{
		GatewayRecord: daos.GatewayRecord{ID: utils.GenerateDatabaseId()}, RequestID: requestID,
		RouteID: route.RouteID, ChannelID: route.ChannelID, Sequence: sequence,
		HTTPStatus: status, DurationMS: duration.Milliseconds(), ErrorCode: errorCode, ErrorSummary: summary,
	}).Error
}

func markRouteSuccess(ctx context.Context, route GatewayRoute) {
	if route.HealthStatus == "healthy" && route.ConsecutiveFailures == 0 {
		return
	}
	result := config.DBConnect.WithContext(ctx).Model(&daos.Channel{}).Where("id = ?", route.ChannelID).
		Updates(map[string]any{"health_status": "healthy", "consecutive_failures": 0, "cooldown_until": nil})
	if result.Error == nil && result.RowsAffected > 0 {
		InvalidateGatewayRouteCache(ctx)
	}
}

func markRouteFailure(ctx context.Context, route GatewayRoute) {
	failures := route.ConsecutiveFailures + 1
	updates := map[string]any{"consecutive_failures": failures}
	if failures >= config.ApplicationConfig.Gateway.FailureThreshold {
		updates["health_status"] = "cooldown"
		updates["cooldown_until"] = time.Now().Add(time.Duration(config.ApplicationConfig.Gateway.CooldownSeconds) * time.Second)
	}
	result := config.DBConnect.WithContext(ctx).Model(&daos.Channel{}).Where("id = ?", route.ChannelID).Updates(updates)
	if result.Error == nil && result.RowsAffected > 0 {
		InvalidateGatewayRouteCache(ctx)
	}
}

func copyGatewayHeaders(destination http.Header, source http.Header) {
	for _, key := range []string{"Content-Type", "Cache-Control", "X-Request-Id"} {
		if value := source.Get(key); value != "" {
			destination.Set(key, value)
		}
	}
}

func parseGatewayUsage(body []byte) (GatewayUsage, bool) {
	var payload struct {
		Usage struct {
			PromptTokens     int64 `json:"prompt_tokens"`
			CompletionTokens int64 `json:"completion_tokens"`
			InputTokens      int64 `json:"input_tokens"`
			OutputTokens     int64 `json:"output_tokens"`
			TotalTokens      int64 `json:"total_tokens"`
		} `json:"usage"`
		Response *struct {
			Usage struct {
				InputTokens  int64 `json:"input_tokens"`
				OutputTokens int64 `json:"output_tokens"`
				TotalTokens  int64 `json:"total_tokens"`
			} `json:"usage"`
		} `json:"response"`
	}
	if json.Unmarshal(body, &payload) != nil {
		return GatewayUsage{}, false
	}
	usage := GatewayUsage{
		PromptTokens: payload.Usage.PromptTokens, CompletionTokens: payload.Usage.CompletionTokens,
		TotalTokens: payload.Usage.TotalTokens,
	}
	if usage.PromptTokens == 0 {
		usage.PromptTokens = payload.Usage.InputTokens
	}
	if usage.CompletionTokens == 0 {
		usage.CompletionTokens = payload.Usage.OutputTokens
	}
	if usage.TotalTokens == 0 && payload.Response != nil {
		usage.PromptTokens = payload.Response.Usage.InputTokens
		usage.CompletionTokens = payload.Response.Usage.OutputTokens
		usage.TotalTokens = payload.Response.Usage.TotalTokens
	}
	if usage.TotalTokens <= 0 {
		return GatewayUsage{}, false
	}
	return usage, true
}

func normalizedGatewayResponse(body []byte, unifiedModel string) []byte {
	var payload map[string]any
	if json.Unmarshal(body, &payload) != nil {
		return body
	}
	if _, ok := payload["model"]; ok {
		payload["model"] = unifiedModel
	}
	if response, ok := payload["response"].(map[string]any); ok {
		if _, hasModel := response["model"]; hasModel {
			response["model"] = unifiedModel
		}
	}
	result, err := json.Marshal(payload)
	if err != nil {
		return body
	}
	return result
}

func streamChatNDJSON(writer http.ResponseWriter, upstreamResponse *http.Response, unifiedModel string) (GatewayUsage, bool, error) {
	writer.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("X-Accel-Buffering", "no")
	writer.WriteHeader(upstreamResponse.StatusCode)
	flusher, _ := writer.(http.Flusher)
	usage := GatewayUsage{}
	foundUsage := false
	scanner := bufio.NewScanner(upstreamResponse.Body)
	scanner.Buffer(make([]byte, 64<<10), 2<<20)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		if bytes.HasPrefix(line, []byte("data:")) {
			line = bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
		} else if bytes.HasPrefix(line, []byte("event:")) || bytes.HasPrefix(line, []byte("id:")) ||
			bytes.HasPrefix(line, []byte("retry:")) || bytes.HasPrefix(line, []byte(":")) {
			continue
		}
		if len(line) == 0 || bytes.Equal(line, []byte("[DONE]")) || !json.Valid(line) {
			continue
		}
		if parsed, ok := parseGatewayUsage(line); ok {
			usage, foundUsage = parsed, true
		}
		chunk := normalizedGatewayResponse(line, unifiedModel)
		if _, err := writer.Write(append(append([]byte(nil), chunk...), '\n')); err != nil {
			return usage, foundUsage, err
		}
		if flusher != nil {
			flusher.Flush()
		}
	}
	return usage, foundUsage, scanner.Err()
}

func (GatewayService) Proxy(ctx context.Context, writer http.ResponseWriter, request *http.Request, principal GatewayPrincipal, model GatewayModel, routes []GatewayRoute, endpoint string, body []byte, streaming bool) error {
	estimate := estimateGatewayTokens(body, config.ApplicationConfig.Gateway.MaxReservedTokens)
	reservation, err := (GatewayService{}).Reserve(ctx, principal, model, strings.TrimPrefix(endpoint, "/"), streaming, estimate)
	if err != nil {
		return err
	}
	writer.Header().Set("X-Request-Id", reservation.RequestID)
	remaining := append([]GatewayRoute(nil), routes...)
	maxAttempts := config.ApplicationConfig.Gateway.MaxAttempts
	if maxAttempts > len(remaining) {
		maxAttempts = len(remaining)
	}
	var finalStatus int
	var finalCode, finalSummary, finalChannel string
	for sequence := 1; sequence <= maxAttempts; sequence++ {
		var route GatewayRoute
		route, remaining, err = selectGatewayRoute(remaining)
		if err != nil {
			break
		}
		finalChannel = route.ChannelID
		upstreamURL, urlErr := gatewayUpstreamURL(route.BaseURL, endpoint)
		apiKey := ""
		var keyErr error
		if route.EncryptedAPIKey != "" {
			apiKey, keyErr = utils.DecryptGatewayCredential(route.EncryptedAPIKey)
		}
		upstreamBody, bodyErr := rewriteGatewayRequest(body, route.UpstreamModel, endpoint, streaming)
		if urlErr != nil || keyErr != nil || bodyErr != nil {
			finalCode, finalSummary = "route_configuration_error", "upstream route configuration is invalid"
			createRouteAttempt(ctx, reservation.RequestID, route, sequence, 0, 0, finalCode, finalSummary)
			markRouteFailure(ctx, route)
			continue
		}
		upstreamRequest, requestErr := http.NewRequestWithContext(ctx, http.MethodPost, upstreamURL, bytes.NewReader(upstreamBody))
		if requestErr != nil {
			finalCode, finalSummary = "upstream_request_error", "failed to create upstream request"
			continue
		}
		upstreamRequest.Header.Set("Content-Type", "application/json")
		if streaming {
			upstreamRequest.Header.Set("Accept", "text/event-stream")
		} else {
			upstreamRequest.Header.Set("Accept", request.Header.Get("Accept"))
		}
		if apiKey != "" {
			upstreamRequest.Header.Set("Authorization", "Bearer "+apiKey)
		}
		timeout := route.TimeoutSeconds
		if timeout <= 0 {
			timeout = config.ApplicationConfig.Gateway.UpstreamTimeoutSeconds
		}
		client := &http.Client{Timeout: time.Duration(timeout) * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
		attemptStarted := time.Now()
		upstreamResponse, requestErr := client.Do(upstreamRequest)
		if requestErr != nil {
			finalCode, finalSummary = "upstream_network_error", "upstream request failed"
			createRouteAttempt(ctx, reservation.RequestID, route, sequence, 0, time.Since(attemptStarted), finalCode, finalSummary)
			markRouteFailure(ctx, route)
			continue
		}
		finalStatus = upstreamResponse.StatusCode
		retryable := upstreamResponse.StatusCode >= 500 || upstreamResponse.StatusCode == http.StatusTooManyRequests
		if retryable {
			_, _ = io.Copy(io.Discard, io.LimitReader(upstreamResponse.Body, 64<<10))
			_ = upstreamResponse.Body.Close()
			finalCode, finalSummary = "upstream_unavailable", fmt.Sprintf("upstream returned HTTP %d", upstreamResponse.StatusCode)
			createRouteAttempt(ctx, reservation.RequestID, route, sequence, upstreamResponse.StatusCode, time.Since(attemptStarted), finalCode, finalSummary)
			markRouteFailure(ctx, route)
			continue
		}
		if upstreamResponse.StatusCode < 200 || upstreamResponse.StatusCode >= 300 {
			_, _ = io.Copy(io.Discard, io.LimitReader(upstreamResponse.Body, 64<<10))
			_ = upstreamResponse.Body.Close()
			finalCode, finalSummary = "upstream_rejected", fmt.Sprintf("upstream returned HTTP %d", upstreamResponse.StatusCode)
			createRouteAttempt(ctx, reservation.RequestID, route, sequence, upstreamResponse.StatusCode, time.Since(attemptStarted), finalCode, finalSummary)
			break
		}

		createRouteAttempt(ctx, reservation.RequestID, route, sequence, upstreamResponse.StatusCode, time.Since(attemptStarted), "", "")
		markRouteSuccess(ctx, route)
		if streaming {
			if endpoint == "/chat/completions" {
				usage, foundUsage, _ := streamChatNDJSON(writer, upstreamResponse, model.Name)
				_ = upstreamResponse.Body.Close()
				if !foundUsage {
					usage.TotalTokens = estimate
				}
				_ = (GatewayService{}).Settle(context.Background(), reservation, usage, true, !foundUsage, upstreamResponse.StatusCode, "", "", route.ChannelID)
				return nil
			}
			copyGatewayHeaders(writer.Header(), upstreamResponse.Header)
			writer.WriteHeader(upstreamResponse.StatusCode)
			flusher, _ := writer.(http.Flusher)
			usage := GatewayUsage{}
			foundUsage := false
			scanner := bufio.NewScanner(upstreamResponse.Body)
			scanner.Buffer(make([]byte, 64<<10), 2<<20)
			for scanner.Scan() {
				line := scanner.Bytes()
				if bytes.HasPrefix(line, []byte("data: ")) && !bytes.Equal(line, []byte("data: [DONE]")) {
					data := bytes.TrimPrefix(line, []byte("data: "))
					if parsed, ok := parseGatewayUsage(data); ok {
						usage, foundUsage = parsed, true
					}
					line = append([]byte("data: "), normalizedGatewayResponse(data, model.Name)...)
				}
				_, _ = writer.Write(append(append([]byte(nil), line...), '\n', '\n'))
				if flusher != nil {
					flusher.Flush()
				}
			}
			_ = upstreamResponse.Body.Close()
			if !foundUsage {
				usage.TotalTokens = estimate
			}
			_ = (GatewayService{}).Settle(context.Background(), reservation, usage, true, !foundUsage, upstreamResponse.StatusCode, "", "", route.ChannelID)
			return nil
		}

		responseBody, readErr := io.ReadAll(io.LimitReader(upstreamResponse.Body, config.ApplicationConfig.Gateway.MaxResponseBodyBytes+1))
		_ = upstreamResponse.Body.Close()
		if readErr != nil || int64(len(responseBody)) > config.ApplicationConfig.Gateway.MaxResponseBodyBytes {
			finalCode, finalSummary = "upstream_response_error", "upstream response is invalid or too large"
			break
		}
		usage, foundUsage := parseGatewayUsage(responseBody)
		if !foundUsage {
			usage.TotalTokens = estimate
		}
		if err = (GatewayService{}).Settle(ctx, reservation, usage, true, !foundUsage, upstreamResponse.StatusCode, "", "", route.ChannelID); err != nil {
			return err
		}
		copyGatewayHeaders(writer.Header(), upstreamResponse.Header)
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(upstreamResponse.StatusCode)
		_, err = writer.Write(normalizedGatewayResponse(responseBody, model.Name))
		return err
	}
	if finalStatus == 0 {
		finalStatus = http.StatusBadGateway
	}
	if finalCode == "" {
		finalCode, finalSummary = "no_available_route", "no upstream route succeeded"
	}
	_ = (GatewayService{}).Settle(context.Background(), reservation, GatewayUsage{}, false, false, finalStatus, finalCode, finalSummary, finalChannel)
	return fmt.Errorf("%s: %s", finalCode, finalSummary)
}

func SortGatewayModels(models []GatewayModel) {
	sort.Slice(models, func(i, j int) bool { return models[i].Name < models[j].Name })
}
