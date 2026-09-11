package dtos

import "time"

type DashboardRange struct {
	Start       time.Time `json:"start" description:"统计开始时间"`
	End         time.Time `json:"end" description:"统计结束时间"`
	Timezone    string    `json:"timezone" description:"统计时区"`
	Granularity string    `json:"granularity" description:"时间粒度"`
}

type DashboardSummary struct {
	TotalRequests     int64    `json:"totalRequests" description:"请求总数"`
	SucceededRequests int64    `json:"succeededRequests" description:"成功请求数"`
	FailedRequests    int64    `json:"failedRequests" description:"最终失败请求数"`
	RejectedRequests  int64    `json:"rejectedRequests" description:"准入拒绝请求数"`
	SuccessRate       *float64 `json:"successRate" description:"成功率"`
	PromptTokens      int64    `json:"promptTokens" description:"输入Token"`
	CompletionTokens  int64    `json:"completionTokens" description:"输出Token"`
	TotalTokens       int64    `json:"totalTokens" description:"Token总量"`
	EstimatedRequests int64    `json:"estimatedRequests" description:"估算用量请求数"`
	EstimatedRatio    *float64 `json:"estimatedRatio" description:"估算用量占比"`
	AverageDurationMS *float64 `json:"averageDurationMs" description:"平均延迟毫秒"`
	P95DurationMS     *float64 `json:"p95DurationMs" description:"P95延迟毫秒"`
}

type DashboardTrendPoint struct {
	Bucket            time.Time `json:"bucket" description:"时间桶"`
	Requests          int64     `json:"requests" description:"请求数"`
	Succeeded         int64     `json:"succeeded" description:"成功请求数"`
	Failed            int64     `json:"failed" description:"失败请求数"`
	Rejected          int64     `json:"rejected" description:"拒绝请求数"`
	PromptTokens      int64     `json:"promptTokens" description:"输入Token"`
	CompletionTokens  int64     `json:"completionTokens" description:"输出Token"`
	TotalTokens       int64     `json:"totalTokens" description:"Token总量"`
	AverageDurationMS *float64  `json:"averageDurationMs" description:"平均延迟毫秒"`
	FailoverRequests  int64     `json:"failoverRequests" description:"故障切换请求数"`
}

type DashboardQuota struct {
	TokenLimit        int64    `json:"tokenLimit" description:"用户Token限额"`
	UsedTokens        int64    `json:"usedTokens" description:"已使用Token"`
	ReservedTokens    int64    `json:"reservedTokens" description:"已预留Token"`
	RemainingTokens   *int64   `json:"remainingTokens" description:"剩余Token"`
	TokenUsageRatio   *float64 `json:"tokenUsageRatio" description:"Token使用比例"`
	RequestLimit      int64    `json:"requestLimit" description:"请求次数限额"`
	UsedRequests      int64    `json:"usedRequests" description:"已使用请求数"`
	ReservedRequests  int64    `json:"reservedRequests" description:"已预留请求数"`
	RemainingRequests *int64   `json:"remainingRequests" description:"剩余请求数"`
	RequestUsageRatio *float64 `json:"requestUsageRatio" description:"请求使用比例"`
}

type DashboardTokenStatus struct {
	Active       int64 `json:"active" description:"有效Token数"`
	Disabled     int64 `json:"disabled" description:"禁用Token数"`
	Expired      int64 `json:"expired" description:"过期Token数"`
	ExpiringSoon int64 `json:"expiringSoon" description:"即将过期Token数"`
}

type DashboardDimensionUsage struct {
	ID            string   `json:"id" description:"对象ID"`
	Name          string   `json:"name" description:"显示名称"`
	Requests      int64    `json:"requests" description:"请求数"`
	TotalTokens   int64    `json:"totalTokens" description:"Token总量"`
	SuccessRate   *float64 `json:"successRate" description:"成功率"`
	P95DurationMS *float64 `json:"p95DurationMs" description:"P95延迟毫秒"`
	UsageRatio    *float64 `json:"usageRatio,omitempty" description:"限额使用比例"`
	Status        string   `json:"status,omitempty" description:"运行状态"`
}

type DashboardErrorBreakdown struct {
	Code  string `json:"code" description:"错误分类"`
	Count int64  `json:"count" description:"错误数量"`
}

type DashboardRecentRequest struct {
	RequestID    string    `json:"requestId" description:"请求ID"`
	CreatedAt    time.Time `json:"createdAt" description:"请求时间"`
	TokenPrefix  string    `json:"tokenPrefix" description:"Token前缀"`
	ModelName    string    `json:"modelName" description:"统一模型"`
	Status       string    `json:"status" description:"请求状态"`
	HTTPStatus   int       `json:"httpStatus" description:"HTTP状态码"`
	TotalTokens  int64     `json:"totalTokens" description:"Token总量"`
	DurationMS   int64     `json:"durationMs" description:"耗时毫秒"`
	ErrorCode    string    `json:"errorCode,omitempty" description:"错误分类"`
	AttemptCount int64     `json:"attemptCount" description:"上游尝试次数"`
}

type DashboardAlert struct {
	Level   string `json:"level" description:"提醒级别"`
	Code    string `json:"code" description:"提醒编码"`
	Title   string `json:"title" description:"提醒标题"`
	Message string `json:"message" description:"提醒内容"`
}

type MyDashboard struct {
	Range           DashboardRange            `json:"range"`
	GeneratedAt     time.Time                 `json:"generatedAt"`
	Quota           DashboardQuota            `json:"quota"`
	TokenStatus     DashboardTokenStatus      `json:"tokenStatus"`
	Summary         DashboardSummary          `json:"summary"`
	PreviousSummary DashboardSummary          `json:"previousSummary"`
	Trend           []DashboardTrendPoint     `json:"trend"`
	ModelUsage      []DashboardDimensionUsage `json:"modelUsage"`
	ErrorBreakdown  []DashboardErrorBreakdown `json:"errorBreakdown"`
	RecentRequests  []DashboardRecentRequest  `json:"recentRequests"`
	Alerts          []DashboardAlert          `json:"alerts"`
}

type DashboardPrincipals struct {
	ActiveUsers        int64 `json:"activeUsers"`
	ActiveTokens       int64 `json:"activeTokens"`
	UsersNearQuota     int64 `json:"usersNearQuota"`
	TokensNearQuota    int64 `json:"tokensNearQuota"`
	TokensExpiringSoon int64 `json:"tokensExpiringSoon"`
}

type DashboardResourceHealth struct {
	ActiveModels          int64 `json:"activeModels"`
	HealthyChannels       int64 `json:"healthyChannels"`
	UnknownChannels       int64 `json:"unknownChannels"`
	CooldownChannels      int64 `json:"cooldownChannels"`
	DisabledChannels      int64 `json:"disabledChannels"`
	MisconfiguredChannels int64 `json:"misconfiguredChannels"`
}

type DashboardRouting struct {
	UpstreamAttempts    int64    `json:"upstreamAttempts"`
	FailoverRequests    int64    `json:"failoverRequests"`
	SuccessfulFailovers int64    `json:"successfulFailovers"`
	FailoverSuccessRate *float64 `json:"failoverSuccessRate"`
}

type AdminDashboard struct {
	Range           DashboardRange            `json:"range"`
	GeneratedAt     time.Time                 `json:"generatedAt"`
	Summary         DashboardSummary          `json:"summary"`
	PreviousSummary DashboardSummary          `json:"previousSummary"`
	Principals      DashboardPrincipals       `json:"principals"`
	ResourceHealth  DashboardResourceHealth   `json:"resourceHealth"`
	Routing         DashboardRouting          `json:"routing"`
	Trend           []DashboardTrendPoint     `json:"trend"`
	ModelUsage      []DashboardDimensionUsage `json:"modelUsage"`
	UserUsage       []DashboardDimensionUsage `json:"userUsage"`
	ChannelUsage    []DashboardDimensionUsage `json:"channelUsage"`
	QuotaRisks      []DashboardAlert          `json:"quotaRisks"`
	RecentFailures  []DashboardRecentRequest  `json:"recentFailures"`
}
