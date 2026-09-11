package services

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sort"
	"time"

	"github.com/efucloud/common"
	"github.com/efucloud/token-router/pkg/config"
	"github.com/efucloud/token-router/pkg/models/daos"
	"github.com/efucloud/token-router/pkg/models/dtos"
	"gorm.io/gorm"
)

type DashboardService struct{}

type dashboardUsageRow struct {
	RequestID        string
	AccountID        string
	APITokenID       string
	TokenPrefix      string
	ModelID          string
	ModelName        string
	ChannelID        string
	Status           string
	HTTPStatus       int
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
	Estimated        bool
	DurationMS       int64
	ErrorCode        string
	CreatedAt        time.Time
}

type dashboardAttemptRow struct {
	RequestID  string
	ChannelID  string
	HTTPStatus int
	DurationMS int64
	CreatedAt  time.Time
}

type dashboardAggregate struct {
	ID          string
	Name        string
	Status      string
	Requests    int64
	Succeeded   int64
	Failed      int64
	TotalTokens int64
	Durations   []int64
	UsageRatio  *float64
}

func dashboardDB(ctx context.Context) *gorm.DB {
	if db, ok := ctx.Value(config.ContextDBTx).(*gorm.DB); ok {
		return db.WithContext(ctx)
	}
	return config.DBConnect.WithContext(ctx)
}

func dashboardError(err error, status int) common.ErrorData {
	return common.ErrorData{Err: err, ResponseCode: status, MsgCode: config.MsgCodeGetRecordFailed}
}

func parseDashboardRange(value string, now time.Time) (dtos.DashboardRange, time.Time, error) {
	duration := 24 * time.Hour
	granularity := "hour"
	switch value {
	case "", "24h":
	case "7d":
		duration = 7 * 24 * time.Hour
		granularity = "day"
	case "30d":
		duration = 30 * 24 * time.Hour
		granularity = "day"
	default:
		return dtos.DashboardRange{}, time.Time{}, fmt.Errorf("unsupported dashboard range %q", value)
	}
	end := now
	start := end.Add(-duration)
	return dtos.DashboardRange{
		Start:       start,
		End:         end,
		Timezone:    now.Location().String(),
		Granularity: granularity,
	}, start.Add(-duration), nil
}

func loadDashboardUsage(db *gorm.DB, start, end time.Time, accountID string) ([]dashboardUsageRow, error) {
	var rows []dashboardUsageRow
	query := db.Table((&daos.UsageLog{}).TableName()).
		Select("request_id, account_id, api_token_id, token_prefix, model_id, model_name, channel_id, status, http_status, prompt_tokens, completion_tokens, total_tokens, estimated, duration_ms, error_code, created_at").
		Where("created_at >= ? AND created_at < ?", start, end)
	if accountID != "" {
		query = query.Where("account_id = ?", accountID)
	}
	return rows, query.Find(&rows).Error
}

func loadDashboardAttempts(db *gorm.DB, start, end time.Time) ([]dashboardAttemptRow, error) {
	var rows []dashboardAttemptRow
	err := db.Table((&daos.RouteAttempt{}).TableName()).
		Select("request_id, channel_id, http_status, duration_ms, created_at").
		Where("created_at >= ? AND created_at < ?", start, end).
		Find(&rows).Error
	return rows, err
}

func ratio(numerator, denominator int64) *float64 {
	if denominator <= 0 {
		return nil
	}
	value := float64(numerator) / float64(denominator)
	return &value
}

func percentile95(values []int64) *float64 {
	if len(values) == 0 {
		return nil
	}
	sorted := append([]int64(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	index := int(math.Ceil(float64(len(sorted))*0.95)) - 1
	value := float64(sorted[index])
	return &value
}

func summarizeDashboard(rows []dashboardUsageRow) dtos.DashboardSummary {
	result := dtos.DashboardSummary{TotalRequests: int64(len(rows))}
	var durations []int64
	for _, row := range rows {
		switch row.Status {
		case "succeeded":
			result.SucceededRequests++
		case "failed":
			result.FailedRequests++
		case "rejected":
			result.RejectedRequests++
		}
		result.PromptTokens += row.PromptTokens
		result.CompletionTokens += row.CompletionTokens
		result.TotalTokens += row.TotalTokens
		if row.Estimated {
			result.EstimatedRequests++
		}
		if row.DurationMS > 0 && (row.Status == "succeeded" || row.Status == "failed") {
			durations = append(durations, row.DurationMS)
		}
	}
	result.SuccessRate = ratio(result.SucceededRequests, result.SucceededRequests+result.FailedRequests)
	result.EstimatedRatio = ratio(result.EstimatedRequests, result.SucceededRequests)
	if len(durations) > 0 {
		var total int64
		for _, duration := range durations {
			total += duration
		}
		average := float64(total) / float64(len(durations))
		result.AverageDurationMS = &average
		result.P95DurationMS = percentile95(durations)
	}
	return result
}

func dashboardBucket(value time.Time, granularity string) time.Time {
	if granularity == "day" {
		return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
	}
	return value.Truncate(time.Hour)
}

func buildDashboardTrend(rows []dashboardUsageRow, timeRange dtos.DashboardRange, failovers map[string]bool) []dtos.DashboardTrendPoint {
	step := time.Hour
	if timeRange.Granularity == "day" {
		step = 24 * time.Hour
	}
	start := dashboardBucket(timeRange.Start, timeRange.Granularity)
	points := make(map[time.Time]*dtos.DashboardTrendPoint)
	durations := make(map[time.Time][]int64)
	for bucket := start; bucket.Before(timeRange.End); bucket = bucket.Add(step) {
		points[bucket] = &dtos.DashboardTrendPoint{Bucket: bucket}
	}
	for _, row := range rows {
		bucket := dashboardBucket(row.CreatedAt, timeRange.Granularity)
		point, ok := points[bucket]
		if !ok {
			continue
		}
		point.Requests++
		switch row.Status {
		case "succeeded":
			point.Succeeded++
		case "failed":
			point.Failed++
		case "rejected":
			point.Rejected++
		}
		point.PromptTokens += row.PromptTokens
		point.CompletionTokens += row.CompletionTokens
		point.TotalTokens += row.TotalTokens
		if row.DurationMS > 0 {
			durations[bucket] = append(durations[bucket], row.DurationMS)
		}
		if failovers[row.RequestID] {
			point.FailoverRequests++
		}
	}
	result := make([]dtos.DashboardTrendPoint, 0, len(points))
	for bucket := start; bucket.Before(timeRange.End); bucket = bucket.Add(step) {
		point := points[bucket]
		if values := durations[bucket]; len(values) > 0 {
			var total int64
			for _, value := range values {
				total += value
			}
			average := float64(total) / float64(len(values))
			point.AverageDurationMS = &average
		}
		result = append(result, *point)
	}
	return result
}

func dimensionResult(values map[string]*dashboardAggregate, limit int) []dtos.DashboardDimensionUsage {
	result := make([]dtos.DashboardDimensionUsage, 0, len(values))
	for _, value := range values {
		result = append(result, dtos.DashboardDimensionUsage{
			ID:            value.ID,
			Name:          value.Name,
			Requests:      value.Requests,
			TotalTokens:   value.TotalTokens,
			SuccessRate:   ratio(value.Succeeded, value.Succeeded+value.Failed),
			P95DurationMS: percentile95(value.Durations),
			UsageRatio:    value.UsageRatio,
			Status:        value.Status,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].TotalTokens == result[j].TotalTokens {
			return result[i].Requests > result[j].Requests
		}
		return result[i].TotalTokens > result[j].TotalTokens
	})
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result
}

func aggregateModels(rows []dashboardUsageRow) []dtos.DashboardDimensionUsage {
	values := make(map[string]*dashboardAggregate)
	for _, row := range rows {
		value := values[row.ModelID]
		if value == nil {
			value = &dashboardAggregate{ID: row.ModelID, Name: row.ModelName}
			values[row.ModelID] = value
		}
		value.Requests++
		value.TotalTokens += row.TotalTokens
		if row.Status == "succeeded" {
			value.Succeeded++
		} else if row.Status == "failed" {
			value.Failed++
		}
		if row.DurationMS > 0 {
			value.Durations = append(value.Durations, row.DurationMS)
		}
	}
	return dimensionResult(values, 10)
}

func recentDashboardRequests(rows []dashboardUsageRow, attempts map[string]int64, failedOnly bool) []dtos.DashboardRecentRequest {
	sorted := append([]dashboardUsageRow(nil), rows...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].CreatedAt.After(sorted[j].CreatedAt) })
	result := make([]dtos.DashboardRecentRequest, 0, 10)
	for _, row := range sorted {
		if failedOnly && row.Status != "failed" {
			continue
		}
		result = append(result, dtos.DashboardRecentRequest{
			RequestID:    row.RequestID,
			CreatedAt:    row.CreatedAt,
			TokenPrefix:  row.TokenPrefix,
			ModelName:    row.ModelName,
			Status:       row.Status,
			HTTPStatus:   row.HTTPStatus,
			TotalTokens:  row.TotalTokens,
			DurationMS:   row.DurationMS,
			ErrorCode:    row.ErrorCode,
			AttemptCount: attempts[row.RequestID],
		})
		if len(result) == 10 {
			break
		}
	}
	return result
}

func quotaDashboard(limit, used, reserved int64) (remaining *int64, usageRatio *float64) {
	if limit <= 0 {
		return nil, nil
	}
	left := limit - used - reserved
	if left < 0 {
		left = 0
	}
	return &left, ratio(used+reserved, limit)
}

func (service DashboardService) GetMyDashboard(ctx context.Context, rangeValue string) (dtos.MyDashboard, common.ErrorData) {
	accountID := config.GetOperatorFromCtx(ctx)
	if accountID == "unknown" || accountID == "" {
		return dtos.MyDashboard{}, dashboardError(fmt.Errorf("authenticated account is missing"), http.StatusUnauthorized)
	}
	return service.getUserDashboard(ctx, accountID, rangeValue)
}

func (service DashboardService) GetUserDashboard(ctx context.Context, accountID, rangeValue string) (dtos.MyDashboard, common.ErrorData) {
	if accountID == "" {
		return dtos.MyDashboard{}, dashboardError(fmt.Errorf("account id is required"), http.StatusBadRequest)
	}
	return service.getUserDashboard(ctx, accountID, rangeValue)
}

func (DashboardService) getUserDashboard(ctx context.Context, accountID, rangeValue string) (dtos.MyDashboard, common.ErrorData) {
	now := time.Now()
	timeRange, previousStart, err := parseDashboardRange(rangeValue, now)
	if err != nil {
		return dtos.MyDashboard{}, dashboardError(err, http.StatusBadRequest)
	}
	db := dashboardDB(ctx)
	var account daos.Account
	if err = db.Where("id = ?", accountID).First(&account).Error; err != nil {
		return dtos.MyDashboard{}, dashboardError(err, http.StatusNotFound)
	}
	rows, err := loadDashboardUsage(db, timeRange.Start, timeRange.End, accountID)
	if err != nil {
		return dtos.MyDashboard{}, dashboardError(err, http.StatusInternalServerError)
	}
	previousRows, err := loadDashboardUsage(db, previousStart, timeRange.Start, accountID)
	if err != nil {
		return dtos.MyDashboard{}, dashboardError(err, http.StatusInternalServerError)
	}

	var tokens []daos.APIToken
	if err = db.Where("account_id = ?", accountID).Find(&tokens).Error; err != nil {
		return dtos.MyDashboard{}, dashboardError(err, http.StatusInternalServerError)
	}
	tokenStatus := dtos.DashboardTokenStatus{}
	for _, token := range tokens {
		if token.Status != "active" {
			tokenStatus.Disabled++
			continue
		}
		if token.ExpiresAt != nil && !token.ExpiresAt.After(now) {
			tokenStatus.Expired++
			continue
		}
		tokenStatus.Active++
		if token.ExpiresAt != nil && token.ExpiresAt.Before(now.Add(7*24*time.Hour)) {
			tokenStatus.ExpiringSoon++
		}
	}

	remainingTokens, tokenRatio := quotaDashboard(account.TokenLimit, account.UsedTokens, account.ReservedTokens)
	remainingRequests, requestRatio := quotaDashboard(account.RequestLimit, account.UsedRequests, account.ReservedRequests)
	result := dtos.MyDashboard{
		Range:       timeRange,
		GeneratedAt: now,
		Quota: dtos.DashboardQuota{
			TokenLimit: account.TokenLimit, UsedTokens: account.UsedTokens, ReservedTokens: account.ReservedTokens,
			RemainingTokens: remainingTokens, TokenUsageRatio: tokenRatio,
			RequestLimit: account.RequestLimit, UsedRequests: account.UsedRequests, ReservedRequests: account.ReservedRequests,
			RemainingRequests: remainingRequests, RequestUsageRatio: requestRatio,
		},
		TokenStatus:     tokenStatus,
		Summary:         summarizeDashboard(rows),
		PreviousSummary: summarizeDashboard(previousRows),
		Trend:           buildDashboardTrend(rows, timeRange, nil),
		ModelUsage:      aggregateModels(rows),
		ErrorBreakdown:  []dtos.DashboardErrorBreakdown{},
		RecentRequests:  recentDashboardRequests(rows, nil, false),
		Alerts:          []dtos.DashboardAlert{},
	}
	errorCounts := make(map[string]int64)
	for _, row := range rows {
		if row.ErrorCode != "" {
			errorCounts[row.ErrorCode]++
		}
	}
	for code, count := range errorCounts {
		result.ErrorBreakdown = append(result.ErrorBreakdown, dtos.DashboardErrorBreakdown{Code: code, Count: count})
	}
	sort.Slice(result.ErrorBreakdown, func(i, j int) bool { return result.ErrorBreakdown[i].Count > result.ErrorBreakdown[j].Count })
	if tokenRatio != nil && *tokenRatio >= 0.8 {
		result.Alerts = append(result.Alerts, dtos.DashboardAlert{Level: "warning", Code: "token_quota_near_limit", Title: "Token 额度接近上限", Message: "用户级 Token 用量已达到 80%"})
	}
	if requestRatio != nil && *requestRatio >= 0.8 {
		result.Alerts = append(result.Alerts, dtos.DashboardAlert{Level: "warning", Code: "request_quota_near_limit", Title: "请求额度接近上限", Message: "用户级请求次数已达到 80%"})
	}
	if tokenStatus.Active == 0 {
		result.Alerts = append(result.Alerts, dtos.DashboardAlert{Level: "info", Code: "no_active_token", Title: "没有可用 API Key", Message: "创建或启用 API Key 后即可调用模型"})
	} else if tokenStatus.ExpiringSoon > 0 {
		result.Alerts = append(result.Alerts, dtos.DashboardAlert{Level: "warning", Code: "token_expiring", Title: "Token 即将过期", Message: fmt.Sprintf("%d 个 Token 将在 7 天内过期", tokenStatus.ExpiringSoon)})
	}
	return result, common.ErrorData{}
}

func (DashboardService) GetAdminDashboard(ctx context.Context, rangeValue string) (dtos.AdminDashboard, common.ErrorData) {
	now := time.Now()
	timeRange, previousStart, err := parseDashboardRange(rangeValue, now)
	if err != nil {
		return dtos.AdminDashboard{}, dashboardError(err, http.StatusBadRequest)
	}
	db := dashboardDB(ctx)
	rows, err := loadDashboardUsage(db, timeRange.Start, timeRange.End, "")
	if err != nil {
		return dtos.AdminDashboard{}, dashboardError(err, http.StatusInternalServerError)
	}
	previousRows, err := loadDashboardUsage(db, previousStart, timeRange.Start, "")
	if err != nil {
		return dtos.AdminDashboard{}, dashboardError(err, http.StatusInternalServerError)
	}
	attemptRows, err := loadDashboardAttempts(db, timeRange.Start, timeRange.End)
	if err != nil {
		return dtos.AdminDashboard{}, dashboardError(err, http.StatusInternalServerError)
	}
	attemptCounts := make(map[string]int64)
	failovers := make(map[string]bool)
	channelAggregates := make(map[string]*dashboardAggregate)
	for _, attempt := range attemptRows {
		attemptCounts[attempt.RequestID]++
		value := channelAggregates[attempt.ChannelID]
		if value == nil {
			value = &dashboardAggregate{ID: attempt.ChannelID, Name: attempt.ChannelID}
			channelAggregates[attempt.ChannelID] = value
		}
		value.Requests++
		if attempt.HTTPStatus >= 200 && attempt.HTTPStatus < 400 {
			value.Succeeded++
		} else {
			value.Failed++
		}
		if attempt.DurationMS > 0 {
			value.Durations = append(value.Durations, attempt.DurationMS)
		}
	}
	for requestID, count := range attemptCounts {
		failovers[requestID] = count > 1
	}
	rowByRequest := make(map[string]dashboardUsageRow)
	activeUsers := make(map[string]bool)
	activeTokens := make(map[string]bool)
	for _, row := range rows {
		rowByRequest[row.RequestID] = row
		if row.Status == "succeeded" || row.Status == "failed" {
			activeUsers[row.AccountID] = true
			if row.APITokenID != systemTokenUsageID {
				activeTokens[row.APITokenID] = true
			}
		}
	}
	routing := dtos.DashboardRouting{UpstreamAttempts: int64(len(attemptRows))}
	for requestID, isFailover := range failovers {
		if !isFailover {
			continue
		}
		routing.FailoverRequests++
		if rowByRequest[requestID].Status == "succeeded" {
			routing.SuccessfulFailovers++
		}
	}
	routing.FailoverSuccessRate = ratio(routing.SuccessfulFailovers, routing.FailoverRequests)

	var accounts []daos.Account
	var tokens []daos.APIToken
	var channels []daos.Channel
	if err = db.Find(&accounts).Error; err != nil {
		return dtos.AdminDashboard{}, dashboardError(err, http.StatusInternalServerError)
	}
	if err = db.Find(&tokens).Error; err != nil {
		return dtos.AdminDashboard{}, dashboardError(err, http.StatusInternalServerError)
	}
	if err = db.Find(&channels).Error; err != nil {
		return dtos.AdminDashboard{}, dashboardError(err, http.StatusInternalServerError)
	}
	var activeModels int64
	if err = db.Model(&daos.AIModel{}).Where("status = ?", "active").Count(&activeModels).Error; err != nil {
		return dtos.AdminDashboard{}, dashboardError(err, http.StatusInternalServerError)
	}

	accountNames := make(map[string]string)
	userAggregates := make(map[string]*dashboardAggregate)
	principals := dtos.DashboardPrincipals{ActiveUsers: int64(len(activeUsers)), ActiveTokens: int64(len(activeTokens))}
	quotaRisks := []dtos.DashboardAlert{}
	for _, account := range accounts {
		name := account.Nickname
		if name == "" {
			name = account.Username
		}
		accountNames[account.ID] = name
		_, tokenUsage := quotaDashboard(account.TokenLimit, account.UsedTokens, account.ReservedTokens)
		_, requestUsage := quotaDashboard(account.RequestLimit, account.UsedRequests, account.ReservedRequests)
		usage := tokenUsage
		if requestUsage != nil && (usage == nil || *requestUsage > *usage) {
			usage = requestUsage
		}
		if usage != nil && *usage >= 0.8 {
			principals.UsersNearQuota++
			quotaRisks = append(quotaRisks, dtos.DashboardAlert{Level: "warning", Code: "user_quota_risk", Title: name, Message: "用户限额使用已达到 80%"})
		}
		userAggregates[account.ID] = &dashboardAggregate{ID: account.ID, Name: name, UsageRatio: usage}
	}
	for _, row := range rows {
		value := userAggregates[row.AccountID]
		if value == nil {
			value = &dashboardAggregate{ID: row.AccountID, Name: accountNames[row.AccountID]}
			userAggregates[row.AccountID] = value
		}
		value.Requests++
		value.TotalTokens += row.TotalTokens
		if row.Status == "succeeded" {
			value.Succeeded++
		} else if row.Status == "failed" {
			value.Failed++
		}
		if row.DurationMS > 0 {
			value.Durations = append(value.Durations, row.DurationMS)
		}
	}
	for _, token := range tokens {
		_, tokenUsage := quotaDashboard(token.TokenLimit, token.UsedTokens, token.ReservedTokens)
		_, requestUsage := quotaDashboard(token.RequestLimit, token.UsedRequests, token.ReservedRequests)
		if (tokenUsage != nil && *tokenUsage >= 0.8) || (requestUsage != nil && *requestUsage >= 0.8) {
			principals.TokensNearQuota++
		}
		if token.Status == "active" && token.ExpiresAt != nil && token.ExpiresAt.After(now) && token.ExpiresAt.Before(now.Add(7*24*time.Hour)) {
			principals.TokensExpiringSoon++
		}
	}

	channelNames := make(map[string]string)
	resourceHealth := dtos.DashboardResourceHealth{ActiveModels: activeModels}
	for _, channel := range channels {
		channelNames[channel.ID] = channel.Name
		if value := channelAggregates[channel.ID]; value != nil {
			value.Name = channel.Name
			value.Status = channel.HealthStatus
		}
		if channel.Status != "enabled" {
			resourceHealth.DisabledChannels++
		} else if channel.EncryptedAPIKey == "" || channel.BaseURL == "" {
			resourceHealth.MisconfiguredChannels++
		} else {
			switch channel.HealthStatus {
			case "healthy":
				resourceHealth.HealthyChannels++
			case "cooldown":
				resourceHealth.CooldownChannels++
			default:
				resourceHealth.UnknownChannels++
			}
		}
	}
	_ = channelNames

	result := dtos.AdminDashboard{
		Range:           timeRange,
		GeneratedAt:     now,
		Summary:         summarizeDashboard(rows),
		PreviousSummary: summarizeDashboard(previousRows),
		Principals:      principals,
		ResourceHealth:  resourceHealth,
		Routing:         routing,
		Trend:           buildDashboardTrend(rows, timeRange, failovers),
		ModelUsage:      aggregateModels(rows),
		UserUsage:       dimensionResult(userAggregates, 10),
		ChannelUsage:    dimensionResult(channelAggregates, 10),
		QuotaRisks:      quotaRisks,
		RecentFailures:  recentDashboardRequests(rows, attemptCounts, true),
	}
	return result, common.ErrorData{}
}
