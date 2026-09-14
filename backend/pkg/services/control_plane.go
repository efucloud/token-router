package services

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/efucloud/common"
	"github.com/efucloud/token-router/pkg/config"
	"github.com/efucloud/token-router/pkg/models/daos"
	"github.com/efucloud/token-router/pkg/models/dtos"
	"github.com/efucloud/token-router/pkg/utils"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

type ControlPlaneService struct{}

type diagnosticRoute struct {
	GatewayRoute
	ChannelName  string
	ProviderName string
}

func controlPlaneError(err error, status int) common.ErrorData {
	return common.ErrorData{Err: err, ResponseCode: status, MsgCode: config.MsgCodeRequestDataInvalid}
}

func controlPlaneWriteError(err error) common.ErrorData {
	status := http.StatusInternalServerError
	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "duplicate") || strings.Contains(lower, "unique constraint") {
		status = http.StatusConflict
	}
	return controlPlaneError(err, status)
}

func validateControlInput(input any) common.ErrorData {
	if err := validator.New().Struct(input); err != nil {
		return controlPlaneError(err, http.StatusBadRequest)
	}
	return common.ErrorData{}
}

func validateControlVersion(version int64) common.ErrorData {
	if version < 1 {
		return controlPlaneError(errors.New("configuration version is required for update"), http.StatusBadRequest)
	}
	return common.ErrorData{}
}

func optimisticUpdateError(db *gorm.DB, result *gorm.DB, table any, id string) common.ErrorData {
	if result.Error != nil {
		return controlPlaneWriteError(result.Error)
	}
	if result.RowsAffected > 0 {
		return common.ErrorData{}
	}
	var count int64
	if err := db.Model(table).Where("id = ?", id).Count(&count).Error; err != nil {
		return controlPlaneError(err, http.StatusInternalServerError)
	}
	if count == 0 {
		return controlPlaneError(gorm.ErrRecordNotFound, http.StatusNotFound)
	}
	return controlPlaneError(errors.New("configuration was changed by another administrator"), http.StatusConflict)
}

func jsonArray(value string) []string {
	result := []string{}
	_ = json.Unmarshal([]byte(value), &result)
	return result
}

func jsonObject(value string) map[string]any {
	result := map[string]any{}
	_ = json.Unmarshal([]byte(value), &result)
	return result
}

type aiModelRow struct {
	daos.AIModel
	RouteCount    int64  `gorm:"column:route_count"`
	RouteID       string `gorm:"column:route_id"`
	RouteStatus   string `gorm:"column:route_status"`
	ChannelID     string `gorm:"column:channel_id"`
	ChannelName   string `gorm:"column:channel_name"`
	ProviderName  string `gorm:"column:provider_name"`
	UpstreamModel string `gorm:"column:upstream_model"`
}

func aiModelDetail(row aiModelRow) dtos.AIModelDetail {
	return dtos.AIModelDetail{
		ID: row.ID, Name: row.Name, DisplayName: row.DisplayName, Description: row.Description,
		Modality: row.Modality, ContextWindow: row.ContextWindow, MaxOutputTokens: row.MaxOutputTokens,
		Capabilities: jsonArray(row.Capabilities), Status: row.Status, RouteCount: row.RouteCount,
		RouteID: row.RouteID, RouteStatus: row.RouteStatus, ChannelID: row.ChannelID,
		ChannelName: row.ChannelName, ProviderName: row.ProviderName, UpstreamModel: row.UpstreamModel,
		Version:   row.Version,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func (ControlPlaneService) ListModels(ctx context.Context, search, status, channelID string) (dtos.AIModelList, common.ErrorData) {
	db := dashboardDB(ctx).Table("ai_model AS m")
	if channelID == "" {
		db = db.Select("m.*, (SELECT COUNT(1) FROM model_route r WHERE r.model_id = m.id) AS route_count")
	} else {
		db = db.Select("m.*, (SELECT COUNT(1) FROM model_route all_routes WHERE all_routes.model_id = m.id) AS route_count, r.id AS route_id, r.status AS route_status, r.channel_id, r.upstream_model, c.name AS channel_name, p.name AS provider_name").
			Joins("JOIN model_route r ON r.model_id = m.id").
			Joins("JOIN channel c ON c.id = r.channel_id").
			Joins("JOIN provider p ON p.id = c.provider_id").
			Where("r.channel_id = ?", channelID)
	}
	if search != "" {
		like := "%" + search + "%"
		db = db.Where("m.name LIKE ? OR m.display_name LIKE ?", like, like)
	}
	if status != "" {
		db = db.Where("m.status = ?", status)
	}
	var rows []aiModelRow
	if err := db.Order("m.created_at DESC").Find(&rows).Error; err != nil {
		return dtos.AIModelList{}, controlPlaneError(err, http.StatusInternalServerError)
	}
	result := dtos.AIModelList{Data: make([]dtos.AIModelDetail, 0, len(rows)), Total: int64(len(rows))}
	for _, row := range rows {
		result.Data = append(result.Data, aiModelDetail(row))
	}
	return result, common.ErrorData{}
}

func (ControlPlaneService) CreateModel(ctx context.Context, input dtos.AIModelInput) (dtos.AIModelDetail, common.ErrorData) {
	if errorData := validateControlInput(input); errorData.IsNotNil() {
		return dtos.AIModelDetail{}, errorData
	}
	capabilities, _ := json.Marshal(input.Capabilities)
	model := daos.AIModel{
		GatewayRecord: daos.GatewayRecord{ID: utils.GenerateDatabaseId()}, Name: strings.TrimSpace(input.Name),
		DisplayName: strings.TrimSpace(input.DisplayName), Description: strings.TrimSpace(input.Description),
		Modality: input.Modality, ContextWindow: input.ContextWindow, MaxOutputTokens: input.MaxOutputTokens,
		Capabilities: string(capabilities), Status: input.Status,
	}
	if err := dashboardDB(ctx).Create(&model).Error; err != nil {
		return dtos.AIModelDetail{}, controlPlaneWriteError(err)
	}
	InvalidateGatewayRouteCache(ctx)
	return aiModelDetail(aiModelRow{AIModel: model}), common.ErrorData{}
}

func (ControlPlaneService) UpdateModel(ctx context.Context, id string, input dtos.AIModelInput) (dtos.AIModelDetail, common.ErrorData) {
	if errorData := validateControlInput(input); errorData.IsNotNil() {
		return dtos.AIModelDetail{}, errorData
	}
	if errorData := validateControlVersion(input.Version); errorData.IsNotNil() {
		return dtos.AIModelDetail{}, errorData
	}
	capabilities, _ := json.Marshal(input.Capabilities)
	db := dashboardDB(ctx)
	result := db.Model(&daos.AIModel{}).Where("id = ? AND version = ?", id, input.Version).Updates(map[string]any{
		"name": strings.TrimSpace(input.Name), "display_name": strings.TrimSpace(input.DisplayName),
		"description": strings.TrimSpace(input.Description), "modality": input.Modality,
		"context_window": input.ContextWindow, "max_output_tokens": input.MaxOutputTokens,
		"capabilities": string(capabilities), "status": input.Status, "version": gorm.Expr("version + 1"),
	})
	if errorData := optimisticUpdateError(db, result, &daos.AIModel{}, id); errorData.IsNotNil() {
		return dtos.AIModelDetail{}, errorData
	}
	var row aiModelRow
	if err := dashboardDB(ctx).Table("ai_model AS m").Select("m.*, (SELECT COUNT(1) FROM model_route r WHERE r.model_id = m.id) AS route_count").Where("m.id = ?", id).First(&row).Error; err != nil {
		return dtos.AIModelDetail{}, controlPlaneError(err, http.StatusInternalServerError)
	}
	InvalidateGatewayRouteCache(ctx)
	return aiModelDetail(row), common.ErrorData{}
}

func (ControlPlaneService) DeleteModel(ctx context.Context, id string) common.ErrorData {
	var count int64
	db := dashboardDB(ctx)
	if err := db.Model(&daos.ModelRoute{}).Where("model_id = ?", id).Count(&count).Error; err != nil {
		return controlPlaneError(err, http.StatusInternalServerError)
	}
	if count > 0 {
		return controlPlaneError(errors.New("model is still referenced by routes"), http.StatusConflict)
	}
	result := db.Where("id = ?", id).Delete(&daos.AIModel{})
	if result.Error != nil {
		return controlPlaneWriteError(result.Error)
	}
	if result.RowsAffected == 0 {
		return controlPlaneError(gorm.ErrRecordNotFound, http.StatusNotFound)
	}
	InvalidateGatewayRouteCache(ctx)
	return common.ErrorData{}
}

type providerRow struct {
	daos.Provider
	ChannelCount int64 `gorm:"column:channel_count"`
}

func providerDetail(row providerRow) dtos.ProviderDetail {
	return dtos.ProviderDetail{
		ID: row.ID, Name: row.Name, Type: row.Type, Status: row.Status, Config: jsonObject(row.Config),
		ChannelCount: row.ChannelCount, Version: row.Version, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func (ControlPlaneService) ListProviders(ctx context.Context, search, status string) (dtos.ProviderList, common.ErrorData) {
	db := dashboardDB(ctx).Table("provider AS p").Select("p.*, (SELECT COUNT(1) FROM channel c WHERE c.provider_id = p.id) AS channel_count")
	if search != "" {
		db = db.Where("p.name LIKE ?", "%"+search+"%")
	}
	if status != "" {
		db = db.Where("p.status = ?", status)
	}
	var rows []providerRow
	if err := db.Order("p.created_at DESC").Find(&rows).Error; err != nil {
		return dtos.ProviderList{}, controlPlaneError(err, http.StatusInternalServerError)
	}
	result := dtos.ProviderList{Data: make([]dtos.ProviderDetail, 0, len(rows)), Total: int64(len(rows))}
	for _, row := range rows {
		result.Data = append(result.Data, providerDetail(row))
	}
	return result, common.ErrorData{}
}

func (ControlPlaneService) CreateProvider(ctx context.Context, input dtos.ProviderInput) (dtos.ProviderDetail, common.ErrorData) {
	if errorData := validateControlInput(input); errorData.IsNotNil() {
		return dtos.ProviderDetail{}, errorData
	}
	configJSON, err := json.Marshal(input.Config)
	if err != nil {
		return dtos.ProviderDetail{}, controlPlaneError(err, http.StatusBadRequest)
	}
	model := daos.Provider{GatewayRecord: daos.GatewayRecord{ID: utils.GenerateDatabaseId()}, Name: strings.TrimSpace(input.Name), Type: input.Type, Status: input.Status, Config: string(configJSON)}
	if err = dashboardDB(ctx).Create(&model).Error; err != nil {
		return dtos.ProviderDetail{}, controlPlaneWriteError(err)
	}
	InvalidateGatewayRouteCache(ctx)
	return providerDetail(providerRow{Provider: model}), common.ErrorData{}
}

func (ControlPlaneService) UpdateProvider(ctx context.Context, id string, input dtos.ProviderInput) (dtos.ProviderDetail, common.ErrorData) {
	if errorData := validateControlInput(input); errorData.IsNotNil() {
		return dtos.ProviderDetail{}, errorData
	}
	if errorData := validateControlVersion(input.Version); errorData.IsNotNil() {
		return dtos.ProviderDetail{}, errorData
	}
	configJSON, err := json.Marshal(input.Config)
	if err != nil {
		return dtos.ProviderDetail{}, controlPlaneError(err, http.StatusBadRequest)
	}
	db := dashboardDB(ctx)
	result := db.Model(&daos.Provider{}).Where("id = ? AND version = ?", id, input.Version).Updates(map[string]any{"name": strings.TrimSpace(input.Name), "type": input.Type, "status": input.Status, "config": string(configJSON), "version": gorm.Expr("version + 1")})
	if errorData := optimisticUpdateError(db, result, &daos.Provider{}, id); errorData.IsNotNil() {
		return dtos.ProviderDetail{}, errorData
	}
	var row providerRow
	if err = dashboardDB(ctx).Table("provider AS p").Select("p.*, (SELECT COUNT(1) FROM channel c WHERE c.provider_id = p.id) AS channel_count").Where("p.id = ?", id).First(&row).Error; err != nil {
		return dtos.ProviderDetail{}, controlPlaneError(err, http.StatusInternalServerError)
	}
	InvalidateGatewayRouteCache(ctx)
	return providerDetail(row), common.ErrorData{}
}

func (ControlPlaneService) DeleteProvider(ctx context.Context, id string) common.ErrorData {
	var count int64
	db := dashboardDB(ctx)
	if err := db.Model(&daos.Channel{}).Where("provider_id = ?", id).Count(&count).Error; err != nil {
		return controlPlaneError(err, http.StatusInternalServerError)
	}
	if count > 0 {
		return controlPlaneError(errors.New("provider is still referenced by channels"), http.StatusConflict)
	}
	result := db.Where("id = ?", id).Delete(&daos.Provider{})
	if result.Error != nil {
		return controlPlaneWriteError(result.Error)
	}
	if result.RowsAffected == 0 {
		return controlPlaneError(gorm.ErrRecordNotFound, http.StatusNotFound)
	}
	InvalidateGatewayRouteCache(ctx)
	return common.ErrorData{}
}

type channelRow struct {
	daos.Channel
	ProviderName string `gorm:"column:provider_name"`
	RouteCount   int64  `gorm:"column:route_count"`
}

func channelDetail(row channelRow) dtos.ChannelDetail {
	return dtos.ChannelDetail{
		ID: row.ID, ProviderID: row.ProviderID, ProviderName: row.ProviderName, Name: row.Name, BaseURL: row.BaseURL,
		CredentialConfigured: row.EncryptedAPIKey != "", Priority: row.Priority, Weight: row.Weight, Status: row.Status,
		TimeoutSeconds: row.TimeoutSeconds, MaxConcurrency: row.MaxConcurrency, HealthStatus: row.HealthStatus,
		CooldownUntil: row.CooldownUntil, ConsecutiveFailures: row.ConsecutiveFailures,
		LastCheckedAt: row.LastCheckedAt, LastLatencyMS: row.LastLatencyMS, LastError: row.LastError,
		RouteCount: row.RouteCount, Version: row.Version, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func channelSelect(db *gorm.DB) *gorm.DB {
	return db.Table("channel AS c").Select("c.*, p.name AS provider_name, (SELECT COUNT(1) FROM model_route r WHERE r.channel_id = c.id) AS route_count").Joins("JOIN provider p ON p.id = c.provider_id")
}

func (ControlPlaneService) ListChannels(ctx context.Context, search, providerID, status string) (dtos.ChannelList, common.ErrorData) {
	db := channelSelect(dashboardDB(ctx))
	if search != "" {
		like := "%" + search + "%"
		db = db.Where("c.name LIKE ? OR c.base_url LIKE ? OR p.name LIKE ?", like, like, like)
	}
	if providerID != "" {
		db = db.Where("c.provider_id = ?", providerID)
	}
	if status != "" {
		db = db.Where("c.status = ?", status)
	}
	var rows []channelRow
	if err := db.Order("c.created_at DESC").Find(&rows).Error; err != nil {
		return dtos.ChannelList{}, controlPlaneError(err, http.StatusInternalServerError)
	}
	result := dtos.ChannelList{Data: make([]dtos.ChannelDetail, 0, len(rows)), Total: int64(len(rows))}
	for _, row := range rows {
		result.Data = append(result.Data, channelDetail(row))
	}
	return result, common.ErrorData{}
}

func validateProvider(ctx context.Context, providerID string) common.ErrorData {
	var count int64
	if err := dashboardDB(ctx).Model(&daos.Provider{}).Where("id = ?", providerID).Count(&count).Error; err != nil {
		return controlPlaneError(err, http.StatusInternalServerError)
	}
	if count == 0 {
		return controlPlaneError(errors.New("provider does not exist"), http.StatusBadRequest)
	}
	return common.ErrorData{}
}

func (ControlPlaneService) CreateChannel(ctx context.Context, input dtos.ChannelInput) (dtos.ChannelDetail, common.ErrorData) {
	if errorData := validateControlInput(input); errorData.IsNotNil() {
		return dtos.ChannelDetail{}, errorData
	}
	if errorData := validateProvider(ctx, input.ProviderID); errorData.IsNotNil() {
		return dtos.ChannelDetail{}, errorData
	}
	encrypted := ""
	var err error
	if strings.TrimSpace(input.APIKey) != "" {
		encrypted, err = utils.EncryptGatewayCredential(input.APIKey)
		if err != nil {
			return dtos.ChannelDetail{}, controlPlaneError(err, http.StatusInternalServerError)
		}
	}
	model := daos.Channel{
		GatewayRecord: daos.GatewayRecord{ID: utils.GenerateDatabaseId()}, ProviderID: input.ProviderID,
		Name: strings.TrimSpace(input.Name), BaseURL: strings.TrimRight(input.BaseURL, "/"), EncryptedAPIKey: encrypted,
		Priority: input.Priority, Weight: input.Weight, Status: input.Status, TimeoutSeconds: input.TimeoutSeconds,
		MaxConcurrency: input.MaxConcurrency, HealthStatus: "unknown",
	}
	if err = dashboardDB(ctx).Create(&model).Error; err != nil {
		return dtos.ChannelDetail{}, controlPlaneWriteError(err)
	}
	var row channelRow
	if err = channelSelect(dashboardDB(ctx)).Where("c.id = ?", model.ID).First(&row).Error; err != nil {
		return dtos.ChannelDetail{}, controlPlaneError(err, http.StatusInternalServerError)
	}
	InvalidateGatewayRouteCache(ctx)
	return channelDetail(row), common.ErrorData{}
}

func (ControlPlaneService) UpdateChannel(ctx context.Context, id string, input dtos.ChannelInput) (dtos.ChannelDetail, common.ErrorData) {
	if errorData := validateControlInput(input); errorData.IsNotNil() {
		return dtos.ChannelDetail{}, errorData
	}
	if errorData := validateControlVersion(input.Version); errorData.IsNotNil() {
		return dtos.ChannelDetail{}, errorData
	}
	if errorData := validateProvider(ctx, input.ProviderID); errorData.IsNotNil() {
		return dtos.ChannelDetail{}, errorData
	}
	values := map[string]any{
		"provider_id": input.ProviderID, "name": strings.TrimSpace(input.Name), "base_url": strings.TrimRight(input.BaseURL, "/"),
		"priority": input.Priority, "weight": input.Weight, "status": input.Status,
		"timeout_seconds": input.TimeoutSeconds, "max_concurrency": input.MaxConcurrency,
		"version": gorm.Expr("version + 1"),
	}
	if strings.TrimSpace(input.APIKey) != "" {
		encrypted, err := utils.EncryptGatewayCredential(input.APIKey)
		if err != nil {
			return dtos.ChannelDetail{}, controlPlaneError(err, http.StatusInternalServerError)
		}
		values["encrypted_api_key"] = encrypted
	}
	db := dashboardDB(ctx)
	result := db.Model(&daos.Channel{}).Where("id = ? AND version = ?", id, input.Version).Updates(values)
	if errorData := optimisticUpdateError(db, result, &daos.Channel{}, id); errorData.IsNotNil() {
		return dtos.ChannelDetail{}, errorData
	}
	var row channelRow
	if err := channelSelect(dashboardDB(ctx)).Where("c.id = ?", id).First(&row).Error; err != nil {
		return dtos.ChannelDetail{}, controlPlaneError(err, http.StatusInternalServerError)
	}
	InvalidateGatewayRouteCache(ctx)
	return channelDetail(row), common.ErrorData{}
}

func (ControlPlaneService) DeleteChannel(ctx context.Context, id string) common.ErrorData {
	var count int64
	db := dashboardDB(ctx)
	if err := db.Model(&daos.ModelRoute{}).Where("channel_id = ?", id).Count(&count).Error; err != nil {
		return controlPlaneError(err, http.StatusInternalServerError)
	}
	if count > 0 {
		return controlPlaneError(errors.New("channel is still referenced by routes"), http.StatusConflict)
	}
	result := db.Where("id = ?", id).Delete(&daos.Channel{})
	if result.Error != nil {
		return controlPlaneWriteError(result.Error)
	}
	if result.RowsAffected == 0 {
		return controlPlaneError(gorm.ErrRecordNotFound, http.StatusNotFound)
	}
	InvalidateGatewayRouteCache(ctx)
	return common.ErrorData{}
}

func truncateControlError(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 500 {
		return value[:500]
	}
	return value
}

func beginChannelProbe(ctx context.Context, channel *daos.Channel, timeout time.Duration) (bool, string, common.ErrorData) {
	now := time.Now()
	db := dashboardDB(ctx)
	switch channel.HealthStatus {
	case "cooldown":
		if channel.CooldownUntil != nil && channel.CooldownUntil.After(now) {
			return false, fmt.Sprintf("channel is cooling down until %s", channel.CooldownUntil.UTC().Format(time.RFC3339)), common.ErrorData{}
		}
		result := db.Model(&daos.Channel{}).
			Where("id = ? AND health_status = ? AND (cooldown_until IS NULL OR cooldown_until <= ?)", channel.ID, "cooldown", now).
			Updates(map[string]any{"health_status": "half_open", "last_checked_at": now})
		if result.Error != nil {
			return false, "", controlPlaneError(result.Error, http.StatusInternalServerError)
		}
		if result.RowsAffected == 0 {
			return false, "channel half-open probe is already running", common.ErrorData{}
		}
		channel.HealthStatus = "half_open"
		channel.LastCheckedAt = &now
		InvalidateGatewayRouteCache(ctx)
		return true, "", common.ErrorData{}
	case "half_open":
		staleBefore := now.Add(-2 * timeout)
		result := db.Model(&daos.Channel{}).
			Where("id = ? AND health_status = ? AND (last_checked_at IS NULL OR last_checked_at <= ?)", channel.ID, "half_open", staleBefore).
			Update("last_checked_at", now)
		if result.Error != nil {
			return false, "", controlPlaneError(result.Error, http.StatusInternalServerError)
		}
		if result.RowsAffected == 0 {
			return false, "channel half-open probe is already running", common.ErrorData{}
		}
		channel.LastCheckedAt = &now
	}
	return true, "", common.ErrorData{}
}

func recordChannelProbe(ctx context.Context, channel daos.Channel, success bool, checkedAt time.Time, latency int64, message string) {
	updates := map[string]any{
		"last_checked_at": checkedAt, "last_latency_ms": latency,
	}
	if success {
		updates["health_status"] = "healthy"
		updates["consecutive_failures"] = 0
		updates["cooldown_until"] = nil
		updates["last_error"] = ""
	} else {
		failures := channel.ConsecutiveFailures + 1
		updates["consecutive_failures"] = failures
		updates["last_error"] = truncateControlError(message)
		if channel.HealthStatus == "half_open" || channel.HealthStatus == "cooldown" || failures >= config.ApplicationConfig.Gateway.FailureThreshold {
			updates["health_status"] = "cooldown"
			updates["cooldown_until"] = checkedAt.Add(time.Duration(config.ApplicationConfig.Gateway.CooldownSeconds) * time.Second)
		} else {
			updates["health_status"] = "unhealthy"
			updates["cooldown_until"] = nil
		}
	}
	result := dashboardDB(ctx).Model(&daos.Channel{}).Where("id = ?", channel.ID).Updates(updates)
	if result.Error == nil && result.RowsAffected > 0 && (channel.HealthStatus != "healthy" || channel.ConsecutiveFailures != 0 || !success) {
		InvalidateGatewayRouteCache(ctx)
	}
}

func (ControlPlaneService) TestChannel(ctx context.Context, id string) (dtos.ChannelTestResult, common.ErrorData) {
	var channel daos.Channel
	if err := dashboardDB(ctx).Where("id = ?", id).First(&channel).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		return dtos.ChannelTestResult{}, controlPlaneError(err, status)
	}
	timeout := time.Duration(channel.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = time.Duration(config.ApplicationConfig.Gateway.UpstreamTimeoutSeconds) * time.Second
	}
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	allowed, message, errorData := beginChannelProbe(ctx, &channel, timeout)
	if errorData.IsNotNil() {
		return dtos.ChannelTestResult{}, errorData
	}
	if !allowed {
		return dtos.ChannelTestResult{Success: false, Message: message, Models: []string{}}, common.ErrorData{}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(channel.BaseURL, "/")+"/models", nil)
	if err != nil {
		recordChannelProbe(ctx, channel, false, time.Now(), 0, "invalid upstream channel URL")
		return dtos.ChannelTestResult{}, controlPlaneError(err, http.StatusBadRequest)
	}
	if channel.EncryptedAPIKey != "" {
		apiKey, decryptErr := utils.DecryptGatewayCredential(channel.EncryptedAPIKey)
		if decryptErr != nil {
			recordChannelProbe(ctx, channel, false, time.Now(), 0, "upstream credential cannot be decrypted")
			return dtos.ChannelTestResult{}, controlPlaneError(decryptErr, http.StatusInternalServerError)
		}
		request.Header.Set(config.AuthHeader, "Bearer "+apiKey)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "Token-Router/1.0")
	startedAt := time.Now()
	response, requestErr := (&http.Client{Timeout: timeout}).Do(request)
	latency := time.Since(startedAt).Milliseconds()
	checkedAt := time.Now()
	if requestErr != nil {
		message := truncateControlError(requestErr.Error())
		recordChannelProbe(ctx, channel, false, checkedAt, latency, message)
		return dtos.ChannelTestResult{Success: false, LatencyMS: latency, Message: message, Models: []string{}}, common.ErrorData{}
	}
	defer response.Body.Close()
	body, readErr := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	result := dtos.ChannelTestResult{Success: response.StatusCode >= 200 && response.StatusCode < 300, HTTPStatus: response.StatusCode, LatencyMS: latency, Models: []string{}}
	if readErr != nil {
		result.Success = false
		result.Message = "failed to read upstream response"
	} else if !result.Success {
		result.Message = fmt.Sprintf("upstream returned HTTP %d", response.StatusCode)
	} else {
		var payload struct {
			Data *[]struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err = json.Unmarshal(body, &payload); err != nil || payload.Data == nil {
			result.Success = false
			result.Message = "upstream response is not an OpenAI-compatible model list"
		} else {
			for _, item := range *payload.Data {
				if item.ID != "" && len(result.Models) < 200 {
					result.Models = append(result.Models, item.ID)
				}
			}
			result.ModelCount = len(*payload.Data)
			result.Message = "connection succeeded"
		}
	}
	recordChannelProbe(ctx, channel, result.Success, checkedAt, latency, result.Message)
	return result, common.ErrorData{}
}

func importedModelModality(name string) string {
	lower := strings.ToLower(name)
	if strings.Contains(lower, "embedding") || strings.Contains(lower, "embed-") || strings.HasPrefix(lower, "embed") {
		return "embedding"
	}
	return "chat"
}

// ImportChannelModels creates enabled unified models and their channel routes.
// Repeating the same import is safe and preserves the status of existing records.
func (ControlPlaneService) ImportChannelModels(ctx context.Context, channelID string, input dtos.ChannelModelImportInput) (dtos.ChannelModelImportResult, common.ErrorData) {
	if errorData := validateControlInput(input); errorData.IsNotNil() {
		return dtos.ChannelModelImportResult{}, errorData
	}
	db := dashboardDB(ctx)
	var channel daos.Channel
	if err := db.Where("id = ?", channelID).First(&channel).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		return dtos.ChannelModelImportResult{}, controlPlaneError(err, status)
	}

	result := dtos.ChannelModelImportResult{Items: []dtos.ChannelModelImportItem{}}
	seen := make(map[string]struct{}, len(input.Models))
	err := db.Transaction(func(tx *gorm.DB) error {
		for _, value := range input.Models {
			upstreamModel := strings.TrimSpace(value)
			if upstreamModel == "" {
				continue
			}
			if _, exists := seen[upstreamModel]; exists {
				continue
			}
			seen[upstreamModel] = struct{}{}
			item := dtos.ChannelModelImportItem{ModelName: upstreamModel, UpstreamModel: upstreamModel}

			var model daos.AIModel
			findModel := tx.Where("name = ?", upstreamModel).First(&model).Error
			if errors.Is(findModel, gorm.ErrRecordNotFound) {
				model = daos.AIModel{
					GatewayRecord: daos.GatewayRecord{ID: utils.GenerateDatabaseId()},
					Name:          upstreamModel, DisplayName: upstreamModel,
					Description: "Imported from channel discovery and enabled by default.",
					Modality:    importedModelModality(upstreamModel), Capabilities: "[]", Status: "active",
				}
				if createErr := tx.Create(&model).Error; createErr != nil {
					return createErr
				}
				item.ModelCreated = true
				result.ModelsCreated++
			} else if findModel != nil {
				return findModel
			}
			item.ModelID = model.ID

			var route daos.ModelRoute
			findRoute := tx.Where("model_id = ? AND channel_id = ? AND upstream_model = ?", model.ID, channelID, upstreamModel).First(&route).Error
			if errors.Is(findRoute, gorm.ErrRecordNotFound) {
				weight := channel.Weight
				if weight < 1 {
					weight = 1
				}
				route = daos.ModelRoute{
					GatewayRecord: daos.GatewayRecord{ID: utils.GenerateDatabaseId()},
					ModelID:       model.ID, ChannelID: channelID, UpstreamModel: upstreamModel,
					Priority: 0, Weight: weight, Status: "enabled",
				}
				if createErr := tx.Create(&route).Error; createErr != nil {
					return createErr
				}
				item.RouteCreated = true
				result.RoutesCreated++
			} else if findRoute != nil {
				return findRoute
			} else {
				result.Skipped++
			}
			result.Items = append(result.Items, item)
		}
		if len(result.Items) == 0 {
			return errors.New("no valid model IDs were provided")
		}
		return nil
	})
	if err != nil {
		return dtos.ChannelModelImportResult{}, controlPlaneWriteError(err)
	}
	if result.ModelsCreated > 0 || result.RoutesCreated > 0 {
		InvalidateGatewayRouteCache(ctx)
	}
	return result, common.ErrorData{}
}

// SyncChannelModels mirrors a channel's successful /models response into the
// routing database. Missing upstream models lose their channel route; a model
// record is deleted only after it has no routes through any other channel.
func (svc ControlPlaneService) SyncChannelModels(ctx context.Context, channelID string) (dtos.ChannelModelSyncResult, common.ErrorData) {
	discovery, errorData := svc.TestChannel(ctx, channelID)
	if errorData.IsNotNil() {
		return dtos.ChannelModelSyncResult{}, errorData
	}
	if !discovery.Success {
		return dtos.ChannelModelSyncResult{}, controlPlaneError(errors.New("channel model discovery failed; database was not changed"), http.StatusBadGateway)
	}
	if discovery.ModelCount > len(discovery.Models) {
		return dtos.ChannelModelSyncResult{}, controlPlaneError(errors.New("channel returned more than 200 models; database was not changed"), http.StatusConflict)
	}

	db := dashboardDB(ctx)
	var channel daos.Channel
	if err := db.Where("id = ?", channelID).First(&channel).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		return dtos.ChannelModelSyncResult{}, controlPlaneError(err, status)
	}

	upstreamModels := make([]string, 0, len(discovery.Models))
	seen := make(map[string]struct{}, len(discovery.Models))
	for _, value := range discovery.Models {
		name := strings.TrimSpace(value)
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		upstreamModels = append(upstreamModels, name)
	}

	result := dtos.ChannelModelSyncResult{Discovered: len(upstreamModels)}
	err := db.Transaction(func(tx *gorm.DB) error {
		var existingRoutes []daos.ModelRoute
		if findErr := tx.Where("channel_id = ?", channelID).Find(&existingRoutes).Error; findErr != nil {
			return findErr
		}

		for _, upstreamModel := range upstreamModels {
			var model daos.AIModel
			findModel := tx.Where("name = ?", upstreamModel).First(&model).Error
			if errors.Is(findModel, gorm.ErrRecordNotFound) {
				modality := importedModelModality(upstreamModel)
				capabilities := "[]"
				if modality == "chat" {
					capabilities = `["streaming"]`
				}
				model = daos.AIModel{
					GatewayRecord: daos.GatewayRecord{ID: utils.GenerateDatabaseId()},
					Name:          upstreamModel,
					DisplayName:   upstreamModel,
					Description:   "Synchronized from upstream channel discovery.",
					Modality:      modality,
					Capabilities:  capabilities,
					Status:        "active",
				}
				if createErr := tx.Create(&model).Error; createErr != nil {
					return createErr
				}
				result.ModelsCreated++
			} else if findModel != nil {
				return findModel
			}

			var route daos.ModelRoute
			findRoute := tx.Where("model_id = ? AND channel_id = ? AND upstream_model = ?", model.ID, channelID, upstreamModel).First(&route).Error
			if errors.Is(findRoute, gorm.ErrRecordNotFound) {
				weight := channel.Weight
				if weight < 1 {
					weight = 1
				}
				route = daos.ModelRoute{
					GatewayRecord: daos.GatewayRecord{ID: utils.GenerateDatabaseId()},
					ModelID:       model.ID,
					ChannelID:     channelID,
					UpstreamModel: upstreamModel,
					Priority:      channel.Priority,
					Weight:        weight,
					Status:        "enabled",
				}
				if createErr := tx.Create(&route).Error; createErr != nil {
					return createErr
				}
				result.RoutesCreated++
			} else if findRoute != nil {
				return findRoute
			} else {
				result.Unchanged++
			}
		}

		orphanCandidates := make(map[string]struct{})
		for _, route := range existingRoutes {
			if _, available := seen[route.UpstreamModel]; available {
				continue
			}
			if deleteErr := tx.Where("id = ?", route.ID).Delete(&daos.ModelRoute{}).Error; deleteErr != nil {
				return deleteErr
			}
			result.RoutesDeleted++
			orphanCandidates[route.ModelID] = struct{}{}
		}

		for modelID := range orphanCandidates {
			var routeCount int64
			if countErr := tx.Model(&daos.ModelRoute{}).Where("model_id = ?", modelID).Count(&routeCount).Error; countErr != nil {
				return countErr
			}
			if routeCount > 0 {
				continue
			}
			deleted := tx.Where("id = ?", modelID).Delete(&daos.AIModel{})
			if deleted.Error != nil {
				return deleted.Error
			}
			if deleted.RowsAffected > 0 {
				result.ModelsDeleted++
			}
		}
		return nil
	})
	if err != nil {
		return dtos.ChannelModelSyncResult{}, controlPlaneWriteError(err)
	}
	if result.ModelsCreated > 0 || result.ModelsDeleted > 0 || result.RoutesCreated > 0 || result.RoutesDeleted > 0 {
		InvalidateGatewayRouteCache(ctx)
	}
	return result, common.ErrorData{}
}

// ProbeEnabledChannels runs the same sanitized connectivity check used by the
// control plane for every enabled channel. It is intentionally best-effort:
// one broken upstream must not prevent other channels from being observed.
func (svc ControlPlaneService) ProbeEnabledChannels(ctx context.Context) (healthy, unhealthy int) {
	var channels []daos.Channel
	if err := dashboardDB(ctx).Where("status = ?", "enabled").Find(&channels).Error; err != nil {
		config.Logger.Errorf("list channels for health probe failed: %v", err)
		return 0, 0
	}
	for _, channel := range channels {
		if channel.HealthStatus == "cooldown" && channel.CooldownUntil != nil && channel.CooldownUntil.After(time.Now()) {
			continue
		}
		result, errorData := svc.TestChannel(ctx, channel.ID)
		if errorData.IsNotNil() || !result.Success {
			unhealthy++
			continue
		}
		healthy++
	}
	return healthy, unhealthy
}

func diagnosticChunk(body []byte) (string, GatewayUsage, bool) {
	var payload struct {
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
		} `json:"choices"`
	}
	_ = json.Unmarshal(body, &payload)
	content := ""
	if len(payload.Choices) > 0 {
		content = payload.Choices[0].Delta.Content
	}
	usage, found := parseGatewayUsage(body)
	return content, usage, found
}

// RunModelDiagnostic performs a transient, administrator-only streaming
// request. Prompt and output are returned to the caller but never persisted.
func (ControlPlaneService) RunModelDiagnostic(ctx context.Context, input dtos.ModelDiagnosticInput) (dtos.ModelDiagnosticResult, common.ErrorData) {
	if errorData := validateControlInput(input); errorData.IsNotNil() {
		return dtos.ModelDiagnosticResult{}, errorData
	}
	var model daos.AIModel
	if err := dashboardDB(ctx).Where("id = ?", input.ModelID).First(&model).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		return dtos.ModelDiagnosticResult{}, controlPlaneError(err, status)
	}
	if model.Modality != "chat" {
		return dtos.ModelDiagnosticResult{}, controlPlaneError(errors.New("only chat models can be diagnosed"), http.StatusBadRequest)
	}

	var rows []diagnosticRoute
	err := dashboardDB(ctx).Table("model_route AS r").Select(`r.id AS route_id, r.channel_id, r.upstream_model,
		CASE WHEN r.priority = 0 THEN c.priority ELSE r.priority END AS priority,
		r.weight, c.base_url, c.encrypted_api_key, c.timeout_seconds, c.consecutive_failures,
		c.name AS channel_name, p.name AS provider_name`).
		Joins("JOIN channel c ON c.id = r.channel_id").Joins("JOIN provider p ON p.id = c.provider_id").
		Where("r.model_id = ? AND r.status = ?", model.ID, "enabled").
		Where("c.status = ? AND p.status = ?", "enabled", "enabled").
		Where("c.health_status NOT IN ?", []string{"cooldown", "half_open"}).Find(&rows).Error
	if err != nil {
		return dtos.ModelDiagnosticResult{}, controlPlaneError(err, http.StatusInternalServerError)
	}
	if len(rows) == 0 {
		return dtos.ModelDiagnosticResult{}, controlPlaneError(errors.New("no enabled route is available for this model"), http.StatusConflict)
	}

	routes := make([]GatewayRoute, 0, len(rows))
	metadata := make(map[string]diagnosticRoute, len(rows))
	for _, row := range rows {
		routes = append(routes, row.GatewayRoute)
		metadata[row.RouteID] = row
	}
	requestID := utils.GenerateDatabaseId()
	startedAt := time.Now()
	result := dtos.ModelDiagnosticResult{
		RequestID: requestID, ModelID: model.ID, ModelName: model.Name,
		Attempts: []dtos.ModelDiagnosticAttempt{}, Estimated: true,
	}
	requestBody, _ := json.Marshal(map[string]any{
		"model": model.Name, "stream": true,
		"stream_options":        map[string]any{"include_usage": true},
		"max_completion_tokens": input.MaxOutputTokens,
		"messages":              []map[string]string{{"role": "user", "content": input.Prompt}},
	})
	estimate := estimateGatewayTokens(requestBody, config.ApplicationConfig.Gateway.MaxReservedTokens)
	remaining := routes
	maxAttempts := config.ApplicationConfig.Gateway.MaxAttempts
	if maxAttempts <= 0 || maxAttempts > len(remaining) {
		maxAttempts = len(remaining)
	}
	for sequence := 1; sequence <= maxAttempts; sequence++ {
		route, rest, selectErr := selectGatewayRoute(remaining)
		remaining = rest
		if selectErr != nil {
			break
		}
		meta := metadata[route.RouteID]
		attempt := dtos.ModelDiagnosticAttempt{
			Sequence: sequence, RouteID: route.RouteID, ChannelID: route.ChannelID,
			ChannelName: meta.ChannelName, ProviderName: meta.ProviderName, UpstreamModel: route.UpstreamModel,
		}
		attemptStarted := time.Now()
		upstreamURL, urlErr := gatewayUpstreamURL(route.BaseURL, "/chat/completions")
		apiKey := ""
		var keyErr error
		if route.EncryptedAPIKey != "" {
			apiKey, keyErr = utils.DecryptGatewayCredential(route.EncryptedAPIKey)
		}
		body, bodyErr := rewriteGatewayRequest(requestBody, route.UpstreamModel, "/chat/completions", true)
		if urlErr != nil || keyErr != nil || bodyErr != nil {
			attempt.ErrorCode, attempt.ErrorSummary = "route_configuration_error", "upstream route configuration is invalid"
			attempt.DurationMS = time.Since(attemptStarted).Milliseconds()
			result.Attempts = append(result.Attempts, attempt)
			markRouteFailure(ctx, route)
			continue
		}
		request, requestErr := http.NewRequestWithContext(ctx, http.MethodPost, upstreamURL, bytes.NewReader(body))
		if requestErr != nil {
			attempt.ErrorCode, attempt.ErrorSummary = "upstream_request_error", "failed to create upstream request"
			result.Attempts = append(result.Attempts, attempt)
			continue
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Accept", "text/event-stream")
		request.Header.Set("X-Request-Id", requestID)
		if apiKey != "" {
			request.Header.Set(config.AuthHeader, "Bearer "+apiKey)
		}
		timeout := route.TimeoutSeconds
		if timeout <= 0 {
			timeout = config.ApplicationConfig.Gateway.UpstreamTimeoutSeconds
		}
		response, requestErr := (&http.Client{Timeout: time.Duration(timeout) * time.Second}).Do(request)
		if requestErr != nil {
			attempt.ErrorCode, attempt.ErrorSummary = "upstream_network_error", "upstream request failed"
			attempt.DurationMS = time.Since(attemptStarted).Milliseconds()
			result.Attempts = append(result.Attempts, attempt)
			markRouteFailure(ctx, route)
			continue
		}
		attempt.HTTPStatus = response.StatusCode
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
			_ = response.Body.Close()
			attempt.ErrorCode = "upstream_rejected"
			attempt.ErrorSummary = fmt.Sprintf("upstream returned HTTP %d", response.StatusCode)
			attempt.DurationMS = time.Since(attemptStarted).Milliseconds()
			result.Attempts = append(result.Attempts, attempt)
			if response.StatusCode >= 500 || response.StatusCode == http.StatusTooManyRequests {
				markRouteFailure(ctx, route)
				continue
			}
			break
		}

		var output strings.Builder
		var usage GatewayUsage
		foundUsage := false
		scanner := bufio.NewScanner(response.Body)
		scanner.Buffer(make([]byte, 64<<10), 2<<20)
		for scanner.Scan() {
			line := scanner.Bytes()
			if !bytes.HasPrefix(line, []byte("data: ")) || bytes.Equal(line, []byte("data: [DONE]")) {
				continue
			}
			content, parsedUsage, found := diagnosticChunk(bytes.TrimPrefix(line, []byte("data: ")))
			if content != "" && result.TTFTMS == 0 {
				result.TTFTMS = time.Since(startedAt).Milliseconds()
			}
			if output.Len()+len(content) <= 64<<10 {
				output.WriteString(content)
			}
			if found {
				usage, foundUsage = parsedUsage, true
			}
		}
		scanErr := scanner.Err()
		_ = response.Body.Close()
		attempt.DurationMS = time.Since(attemptStarted).Milliseconds()
		if scanErr != nil {
			attempt.ErrorCode, attempt.ErrorSummary = "upstream_stream_error", "upstream stream ended unexpectedly"
			result.Attempts = append(result.Attempts, attempt)
			markRouteFailure(ctx, route)
			continue
		}
		result.Attempts = append(result.Attempts, attempt)
		markRouteSuccess(ctx, route)
		if !foundUsage {
			usage.TotalTokens = estimate
		}
		result.Success = true
		result.OutputText = output.String()
		result.PromptTokens = usage.PromptTokens
		result.CompletionTokens = usage.CompletionTokens
		result.TotalTokens = usage.TotalTokens
		result.Estimated = !foundUsage
		result.FinalChannelID = route.ChannelID
		result.FinalChannelName = meta.ChannelName
		result.DurationMS = time.Since(startedAt).Milliseconds()
		result.Message = "diagnostic completed"
		return result, common.ErrorData{}
	}
	result.DurationMS = time.Since(startedAt).Milliseconds()
	result.Message = "all upstream routes failed"
	return result, common.ErrorData{}
}

type modelRouteRow struct {
	daos.ModelRoute
	ModelName    string `gorm:"column:model_name"`
	ChannelName  string `gorm:"column:channel_name"`
	ProviderName string `gorm:"column:provider_name"`
}

func modelRouteDetail(row modelRouteRow) dtos.ModelRouteDetail {
	return dtos.ModelRouteDetail{
		ID: row.ID, ModelID: row.ModelID, ModelName: row.ModelName, ChannelID: row.ChannelID,
		ChannelName: row.ChannelName, ProviderName: row.ProviderName, UpstreamModel: row.UpstreamModel,
		Priority: row.Priority, Weight: row.Weight, Status: row.Status, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		Version: row.Version,
	}
}

func routeSelect(db *gorm.DB) *gorm.DB {
	return db.Table("model_route AS r").Select("r.*, m.name AS model_name, c.name AS channel_name, p.name AS provider_name").Joins("JOIN ai_model m ON m.id = r.model_id").Joins("JOIN channel c ON c.id = r.channel_id").Joins("JOIN provider p ON p.id = c.provider_id")
}

func (ControlPlaneService) ListRoutes(ctx context.Context, modelID, channelID, status string) (dtos.ModelRouteList, common.ErrorData) {
	db := routeSelect(dashboardDB(ctx))
	if modelID != "" {
		db = db.Where("r.model_id = ?", modelID)
	}
	if channelID != "" {
		db = db.Where("r.channel_id = ?", channelID)
	}
	if status != "" {
		db = db.Where("r.status = ?", status)
	}
	var rows []modelRouteRow
	if err := db.Order("m.name ASC, r.priority DESC, r.weight DESC").Find(&rows).Error; err != nil {
		return dtos.ModelRouteList{}, controlPlaneError(err, http.StatusInternalServerError)
	}
	result := dtos.ModelRouteList{Data: make([]dtos.ModelRouteDetail, 0, len(rows)), Total: int64(len(rows))}
	for _, row := range rows {
		result.Data = append(result.Data, modelRouteDetail(row))
	}
	return result, common.ErrorData{}
}

func validateRouteRefs(ctx context.Context, modelID, channelID string) common.ErrorData {
	var modelCount, channelCount int64
	db := dashboardDB(ctx)
	if err := db.Model(&daos.AIModel{}).Where("id = ?", modelID).Count(&modelCount).Error; err != nil {
		return controlPlaneError(err, http.StatusInternalServerError)
	}
	if err := db.Model(&daos.Channel{}).Where("id = ?", channelID).Count(&channelCount).Error; err != nil {
		return controlPlaneError(err, http.StatusInternalServerError)
	}
	if modelCount == 0 || channelCount == 0 {
		return controlPlaneError(errors.New("model or channel does not exist"), http.StatusBadRequest)
	}
	return common.ErrorData{}
}

func (ControlPlaneService) CreateRoute(ctx context.Context, input dtos.ModelRouteInput) (dtos.ModelRouteDetail, common.ErrorData) {
	if errorData := validateControlInput(input); errorData.IsNotNil() {
		return dtos.ModelRouteDetail{}, errorData
	}
	if errorData := validateRouteRefs(ctx, input.ModelID, input.ChannelID); errorData.IsNotNil() {
		return dtos.ModelRouteDetail{}, errorData
	}
	model := daos.ModelRoute{GatewayRecord: daos.GatewayRecord{ID: utils.GenerateDatabaseId()}, ModelID: input.ModelID, ChannelID: input.ChannelID, UpstreamModel: strings.TrimSpace(input.UpstreamModel), Priority: input.Priority, Weight: input.Weight, Status: input.Status}
	if err := dashboardDB(ctx).Create(&model).Error; err != nil {
		return dtos.ModelRouteDetail{}, controlPlaneWriteError(err)
	}
	var row modelRouteRow
	if err := routeSelect(dashboardDB(ctx)).Where("r.id = ?", model.ID).First(&row).Error; err != nil {
		return dtos.ModelRouteDetail{}, controlPlaneError(err, http.StatusInternalServerError)
	}
	InvalidateGatewayRouteCache(ctx)
	return modelRouteDetail(row), common.ErrorData{}
}

func (ControlPlaneService) UpdateRoute(ctx context.Context, id string, input dtos.ModelRouteInput) (dtos.ModelRouteDetail, common.ErrorData) {
	if errorData := validateControlInput(input); errorData.IsNotNil() {
		return dtos.ModelRouteDetail{}, errorData
	}
	if errorData := validateControlVersion(input.Version); errorData.IsNotNil() {
		return dtos.ModelRouteDetail{}, errorData
	}
	if errorData := validateRouteRefs(ctx, input.ModelID, input.ChannelID); errorData.IsNotNil() {
		return dtos.ModelRouteDetail{}, errorData
	}
	db := dashboardDB(ctx)
	result := db.Model(&daos.ModelRoute{}).Where("id = ? AND version = ?", id, input.Version).Updates(map[string]any{"model_id": input.ModelID, "channel_id": input.ChannelID, "upstream_model": strings.TrimSpace(input.UpstreamModel), "priority": input.Priority, "weight": input.Weight, "status": input.Status, "version": gorm.Expr("version + 1")})
	if errorData := optimisticUpdateError(db, result, &daos.ModelRoute{}, id); errorData.IsNotNil() {
		return dtos.ModelRouteDetail{}, errorData
	}
	var row modelRouteRow
	if err := routeSelect(dashboardDB(ctx)).Where("r.id = ?", id).First(&row).Error; err != nil {
		return dtos.ModelRouteDetail{}, controlPlaneError(err, http.StatusInternalServerError)
	}
	InvalidateGatewayRouteCache(ctx)
	return modelRouteDetail(row), common.ErrorData{}
}

func (ControlPlaneService) DeleteRoute(ctx context.Context, id string) common.ErrorData {
	result := dashboardDB(ctx).Where("id = ?", id).Delete(&daos.ModelRoute{})
	if result.Error != nil {
		return controlPlaneWriteError(result.Error)
	}
	if result.RowsAffected == 0 {
		return controlPlaneError(gorm.ErrRecordNotFound, http.StatusNotFound)
	}
	InvalidateGatewayRouteCache(ctx)
	return common.ErrorData{}
}

func (ControlPlaneService) ListUsage(ctx context.Context, current, pageSize int, requestID, accountID, modelID, channelID, status string, start, end *time.Time) (dtos.UsageLogList, common.ErrorData) {
	if current < 1 {
		current = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	db := dashboardDB(ctx).Table("usage_log AS u").Joins("LEFT JOIN account a ON a.id = u.account_id").Joins("LEFT JOIN channel c ON c.id = u.channel_id")
	if requestID != "" {
		db = db.Where("u.request_id LIKE ?", "%"+requestID+"%")
	}
	if accountID != "" {
		db = db.Where("u.account_id = ?", accountID)
	}
	if modelID != "" {
		db = db.Where("u.model_id = ?", modelID)
	}
	if channelID != "" {
		db = db.Where("u.channel_id = ?", channelID)
	}
	if status != "" {
		db = db.Where("u.status = ?", status)
	}
	if start != nil {
		db = db.Where("u.created_at >= ?", *start)
	}
	if end != nil {
		db = db.Where("u.created_at <= ?", *end)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return dtos.UsageLogList{}, controlPlaneError(err, http.StatusInternalServerError)
	}
	var data []dtos.UsageLogDetail
	if err := db.Select("u.id, u.request_id, u.account_id, COALESCE(a.username, '') AS username, u.api_token_id, u.token_prefix, u.model_id, u.model_name, u.channel_id, COALESCE(c.name, '') AS channel_name, u.endpoint, u.streaming, u.prompt_tokens, u.completion_tokens, u.total_tokens, u.estimated, u.status, u.http_status, u.duration_ms, u.error_code, u.error_summary, u.created_at").Order("u.created_at DESC").Offset((current - 1) * pageSize).Limit(pageSize).Scan(&data).Error; err != nil {
		return dtos.UsageLogList{}, controlPlaneError(err, http.StatusInternalServerError)
	}
	if data == nil {
		data = []dtos.UsageLogDetail{}
	}
	return dtos.UsageLogList{Data: data, Total: total}, common.ErrorData{}
}

func (ControlPlaneService) ListAttempts(ctx context.Context, current, pageSize int, requestID, channelID string) (dtos.RouteAttemptList, common.ErrorData) {
	if current < 1 {
		current = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	db := dashboardDB(ctx).Table("route_attempt AS r").Joins("LEFT JOIN channel c ON c.id = r.channel_id")
	if requestID != "" {
		db = db.Where("r.request_id = ?", requestID)
	}
	if channelID != "" {
		db = db.Where("r.channel_id = ?", channelID)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return dtos.RouteAttemptList{}, controlPlaneError(err, http.StatusInternalServerError)
	}
	var data []dtos.RouteAttemptDetail
	if err := db.Select("r.id, r.request_id, r.route_id, r.channel_id, COALESCE(c.name, '') AS channel_name, r.sequence, r.http_status, r.duration_ms, r.error_code, r.error_summary, r.created_at").Order("r.created_at DESC, r.sequence ASC").Offset((current - 1) * pageSize).Limit(pageSize).Scan(&data).Error; err != nil {
		return dtos.RouteAttemptList{}, controlPlaneError(err, http.StatusInternalServerError)
	}
	if data == nil {
		data = []dtos.RouteAttemptDetail{}
	}
	return dtos.RouteAttemptList{Data: data, Total: total}, common.ErrorData{}
}

type auditLogRow struct {
	daos.AuditLog
	OperatorUsername string `gorm:"column:operator_username"`
}

func (ControlPlaneService) ListAuditLogs(ctx context.Context, current, pageSize int, operatorID, action, resourceType, result string) (dtos.AuditLogList, common.ErrorData) {
	if current < 1 {
		current = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	db := dashboardDB(ctx).Table("audit_log AS l").Joins("LEFT JOIN account a ON a.id = l.operator_id")
	if operatorID != "" {
		db = db.Where("l.operator_id = ?", operatorID)
	}
	if action != "" {
		db = db.Where("l.action = ?", action)
	}
	if resourceType != "" {
		db = db.Where("l.resource_type = ?", resourceType)
	}
	if result != "" {
		db = db.Where("l.result = ?", result)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return dtos.AuditLogList{}, controlPlaneError(err, http.StatusInternalServerError)
	}
	var rows []auditLogRow
	if err := db.Select("l.*, COALESCE(a.username, '') AS operator_username").Order("l.created_at DESC").Offset((current - 1) * pageSize).Limit(pageSize).Scan(&rows).Error; err != nil {
		return dtos.AuditLogList{}, controlPlaneError(err, http.StatusInternalServerError)
	}
	data := make([]dtos.AuditLogDetail, 0, len(rows))
	for _, row := range rows {
		summary := map[string]string{}
		_ = json.Unmarshal([]byte(row.Summary), &summary)
		data = append(data, dtos.AuditLogDetail{
			ID: row.ID, OperatorID: row.OperatorID, OperatorUsername: row.OperatorUsername,
			Action: row.Action, ResourceType: row.ResourceType, ResourceID: row.ResourceID,
			Method: row.Method, RoutePath: row.RoutePath, Summary: summary, Result: row.Result,
			StatusCode: row.StatusCode, RequestID: row.RequestID, RemoteIP: row.RemoteIP, CreatedAt: row.CreatedAt,
		})
	}
	return dtos.AuditLogList{Data: data, Total: total}, common.ErrorData{}
}
