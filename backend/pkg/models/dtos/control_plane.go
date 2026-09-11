package dtos

import "time"

type AIModelInput struct {
	Name            string   `json:"name" validate:"required,max=255" description:"对内发布的统一模型名称"`
	DisplayName     string   `json:"displayName" validate:"required,max=255" description:"模型显示名称"`
	Description     string   `json:"description" validate:"max=1000" description:"模型说明"`
	Modality        string   `json:"modality" validate:"oneof=chat embedding" enum:"chat|embedding" description:"模型模态"`
	ContextWindow   int64    `json:"contextWindow" validate:"gte=0" description:"上下文窗口"`
	MaxOutputTokens int64    `json:"maxOutputTokens" validate:"gte=0" description:"最大输出Token"`
	Capabilities    []string `json:"capabilities" description:"能力标签"`
	Status          string   `json:"status" validate:"oneof=active disabled" enum:"active|disabled" description:"发布状态"`
	Version         int64    `json:"version" validate:"gte=0" description:"配置版本；创建时为0"`
}

type AIModelDetail struct {
	ID              string    `json:"id" description:"记录ID"`
	Name            string    `json:"name" description:"统一模型名称"`
	DisplayName     string    `json:"displayName" description:"模型显示名称"`
	Description     string    `json:"description" description:"模型说明"`
	Modality        string    `json:"modality" description:"模型模态"`
	ContextWindow   int64     `json:"contextWindow" description:"上下文窗口"`
	MaxOutputTokens int64     `json:"maxOutputTokens" description:"最大输出Token"`
	Capabilities    []string  `json:"capabilities" description:"能力标签"`
	Status          string    `json:"status" description:"发布状态"`
	RouteCount      int64     `json:"routeCount" description:"路由数量"`
	RouteID         string    `json:"routeId,omitempty" description:"按渠道筛选时的路由ID"`
	RouteStatus     string    `json:"routeStatus,omitempty" description:"按渠道筛选时的路由状态"`
	ChannelID       string    `json:"channelId,omitempty" description:"按渠道筛选时的渠道ID"`
	ChannelName     string    `json:"channelName,omitempty" description:"按渠道筛选时的渠道名称"`
	ProviderName    string    `json:"providerName,omitempty" description:"按渠道筛选时的供应商名称"`
	UpstreamModel   string    `json:"upstreamModel,omitempty" description:"按渠道筛选时的上游模型ID"`
	Version         int64     `json:"version" description:"配置版本"`
	CreatedAt       time.Time `json:"createdAt" description:"创建时间"`
	UpdatedAt       time.Time `json:"updatedAt" description:"更新时间"`
}

type AIModelList struct {
	Data  []AIModelDetail `json:"data"`
	Total int64           `json:"total"`
}

type ProviderInput struct {
	Name    string         `json:"name" validate:"required,max=255" description:"供应商名称"`
	Type    string         `json:"type" validate:"oneof=openai-compatible" enum:"openai-compatible" description:"适配器类型"`
	Status  string         `json:"status" validate:"oneof=enabled disabled" enum:"enabled|disabled" description:"状态"`
	Config  map[string]any `json:"config" description:"非敏感适配器配置"`
	Version int64          `json:"version" validate:"gte=0" description:"配置版本；创建时为0"`
}

type ProviderDetail struct {
	ID           string         `json:"id" description:"记录ID"`
	Name         string         `json:"name" description:"供应商名称"`
	Type         string         `json:"type" description:"适配器类型"`
	Status       string         `json:"status" description:"状态"`
	Config       map[string]any `json:"config" description:"非敏感适配器配置"`
	ChannelCount int64          `json:"channelCount" description:"渠道数量"`
	Version      int64          `json:"version" description:"配置版本"`
	CreatedAt    time.Time      `json:"createdAt" description:"创建时间"`
	UpdatedAt    time.Time      `json:"updatedAt" description:"更新时间"`
}

type ProviderList struct {
	Data  []ProviderDetail `json:"data"`
	Total int64            `json:"total"`
}

type ChannelInput struct {
	ProviderID     string `json:"providerId" validate:"required" description:"供应商ID"`
	Name           string `json:"name" validate:"required,max=255" description:"渠道名称"`
	BaseURL        string `json:"baseUrl" validate:"required,url,max=1000" description:"上游Base URL"`
	APIKey         string `json:"apiKey" description:"上游密钥；更新时留空表示保持不变"`
	Priority       int    `json:"priority" validate:"gte=0" description:"渠道默认优先级"`
	Weight         int    `json:"weight" validate:"gte=1" description:"渠道默认权重"`
	Status         string `json:"status" validate:"oneof=enabled disabled" enum:"enabled|disabled" description:"状态"`
	TimeoutSeconds int    `json:"timeoutSeconds" validate:"gte=0,lte=600" description:"上游超时秒数；0使用系统默认"`
	MaxConcurrency int    `json:"maxConcurrency" validate:"gte=0" description:"最大并发；0表示不限制"`
	Version        int64  `json:"version" validate:"gte=0" description:"配置版本；创建时为0"`
}

type ChannelDetail struct {
	ID                   string     `json:"id" description:"记录ID"`
	ProviderID           string     `json:"providerId" description:"供应商ID"`
	ProviderName         string     `json:"providerName" description:"供应商名称"`
	Name                 string     `json:"name" description:"渠道名称"`
	BaseURL              string     `json:"baseUrl" description:"上游Base URL"`
	CredentialConfigured bool       `json:"credentialConfigured" description:"是否已配置密钥"`
	Priority             int        `json:"priority" description:"默认优先级"`
	Weight               int        `json:"weight" description:"默认权重"`
	Status               string     `json:"status" description:"状态"`
	TimeoutSeconds       int        `json:"timeoutSeconds" description:"上游超时秒数"`
	MaxConcurrency       int        `json:"maxConcurrency" description:"最大并发"`
	HealthStatus         string     `json:"healthStatus" description:"健康状态"`
	CooldownUntil        *time.Time `json:"cooldownUntil" description:"冷却截止时间"`
	ConsecutiveFailures  int        `json:"consecutiveFailures" description:"连续失败数"`
	LastCheckedAt        *time.Time `json:"lastCheckedAt" description:"最近检查时间"`
	LastLatencyMS        int64      `json:"lastLatencyMs" description:"最近检查延迟毫秒"`
	LastError            string     `json:"lastError" description:"最近检查错误摘要"`
	RouteCount           int64      `json:"routeCount" description:"路由数量"`
	Version              int64      `json:"version" description:"配置版本"`
	CreatedAt            time.Time  `json:"createdAt" description:"创建时间"`
	UpdatedAt            time.Time  `json:"updatedAt" description:"更新时间"`
}

type ChannelList struct {
	Data  []ChannelDetail `json:"data"`
	Total int64           `json:"total"`
}

type ChannelTestResult struct {
	Success    bool     `json:"success" description:"连接是否成功"`
	HTTPStatus int      `json:"httpStatus" description:"上游HTTP状态"`
	LatencyMS  int64    `json:"latencyMs" description:"延迟毫秒"`
	ModelCount int      `json:"modelCount" description:"发现的模型数量"`
	Models     []string `json:"models" description:"发现的模型ID"`
	Message    string   `json:"message" description:"检查结果说明"`
}

type ChannelModelImportInput struct {
	Models []string `json:"models" validate:"required,min=1,max=200,dive,required,max=255" description:"要导入的上游模型ID"`
}

type ChannelModelImportItem struct {
	ModelID       string `json:"modelId" description:"统一模型ID"`
	ModelName     string `json:"modelName" description:"统一模型名称"`
	UpstreamModel string `json:"upstreamModel" description:"上游模型ID"`
	ModelCreated  bool   `json:"modelCreated" description:"是否新建统一模型"`
	RouteCreated  bool   `json:"routeCreated" description:"是否新建路由"`
}

type ChannelModelImportResult struct {
	ModelsCreated int                      `json:"modelsCreated" description:"新建的统一模型数"`
	RoutesCreated int                      `json:"routesCreated" description:"新建的路由数"`
	Skipped       int                      `json:"skipped" description:"已存在并跳过的映射数"`
	Items         []ChannelModelImportItem `json:"items" description:"导入结果"`
}

type ChannelModelSyncInput struct{}

type ChannelModelSyncResult struct {
	Discovered    int `json:"discovered" description:"上游发现的模型数"`
	ModelsCreated int `json:"modelsCreated" description:"新建的统一模型数"`
	RoutesCreated int `json:"routesCreated" description:"新建的渠道路由数"`
	RoutesDeleted int `json:"routesDeleted" description:"删除的失效渠道路由数"`
	ModelsDeleted int `json:"modelsDeleted" description:"删除的孤立模型数"`
	Unchanged     int `json:"unchanged" description:"无需变更的模型数"`
}

type ModelRouteInput struct {
	ModelID       string `json:"modelId" validate:"required" description:"统一模型ID"`
	ChannelID     string `json:"channelId" validate:"required" description:"渠道ID"`
	UpstreamModel string `json:"upstreamModel" validate:"required,max=255" description:"上游模型名称"`
	Priority      int    `json:"priority" validate:"gte=0" description:"路由优先级"`
	Weight        int    `json:"weight" validate:"gte=1" description:"同优先级权重"`
	Status        string `json:"status" validate:"oneof=enabled disabled" enum:"enabled|disabled" description:"状态"`
	Version       int64  `json:"version" validate:"gte=0" description:"配置版本；创建时为0"`
}

type ModelRouteDetail struct {
	ID            string    `json:"id" description:"记录ID"`
	ModelID       string    `json:"modelId" description:"统一模型ID"`
	ModelName     string    `json:"modelName" description:"统一模型名称"`
	ChannelID     string    `json:"channelId" description:"渠道ID"`
	ChannelName   string    `json:"channelName" description:"渠道名称"`
	ProviderName  string    `json:"providerName" description:"供应商名称"`
	UpstreamModel string    `json:"upstreamModel" description:"上游模型名称"`
	Priority      int       `json:"priority" description:"路由优先级"`
	Weight        int       `json:"weight" description:"权重"`
	Status        string    `json:"status" description:"状态"`
	Version       int64     `json:"version" description:"配置版本"`
	CreatedAt     time.Time `json:"createdAt" description:"创建时间"`
	UpdatedAt     time.Time `json:"updatedAt" description:"更新时间"`
}

type ModelRouteList struct {
	Data  []ModelRouteDetail `json:"data"`
	Total int64              `json:"total"`
}

type UsageLogDetail struct {
	ID               string    `json:"id" description:"记录ID"`
	RequestID        string    `json:"requestId" description:"请求ID"`
	AccountID        string    `json:"accountId" description:"用户ID"`
	Username         string    `json:"username" description:"用户名"`
	APITokenID       string    `json:"apiTokenId" description:"API Key ID"`
	TokenPrefix      string    `json:"tokenPrefix" description:"Token前缀"`
	ModelID          string    `json:"modelId" description:"统一模型ID"`
	ModelName        string    `json:"modelName" description:"统一模型名称"`
	ChannelID        string    `json:"channelId" description:"渠道ID"`
	ChannelName      string    `json:"channelName" description:"渠道名称"`
	Endpoint         string    `json:"endpoint" description:"数据面端点"`
	Streaming        bool      `json:"streaming" description:"是否流式"`
	PromptTokens     int64     `json:"promptTokens" description:"输入Token"`
	CompletionTokens int64     `json:"completionTokens" description:"输出Token"`
	TotalTokens      int64     `json:"totalTokens" description:"总Token"`
	Estimated        bool      `json:"estimated" description:"用量是否为估算"`
	Status           string    `json:"status" description:"状态"`
	HTTPStatus       int       `json:"httpStatus" description:"HTTP状态"`
	DurationMS       int64     `json:"durationMs" description:"耗时毫秒"`
	ErrorCode        string    `json:"errorCode" description:"错误码"`
	ErrorSummary     string    `json:"errorSummary" description:"错误摘要"`
	CreatedAt        time.Time `json:"createdAt" description:"创建时间"`
}

type UsageLogList struct {
	Data  []UsageLogDetail `json:"data"`
	Total int64            `json:"total"`
}

type RouteAttemptDetail struct {
	ID           string    `json:"id" description:"记录ID"`
	RequestID    string    `json:"requestId" description:"请求ID"`
	RouteID      string    `json:"routeId" description:"路由ID"`
	ChannelID    string    `json:"channelId" description:"渠道ID"`
	ChannelName  string    `json:"channelName" description:"渠道名称"`
	Sequence     int       `json:"sequence" description:"尝试序号"`
	HTTPStatus   int       `json:"httpStatus" description:"HTTP状态"`
	DurationMS   int64     `json:"durationMs" description:"耗时毫秒"`
	ErrorCode    string    `json:"errorCode" description:"错误码"`
	ErrorSummary string    `json:"errorSummary" description:"错误摘要"`
	CreatedAt    time.Time `json:"createdAt" description:"创建时间"`
}

type RouteAttemptList struct {
	Data  []RouteAttemptDetail `json:"data"`
	Total int64                `json:"total"`
}

type AuditLogDetail struct {
	ID               string            `json:"id" description:"记录ID"`
	OperatorID       string            `json:"operatorId" description:"操作者ID"`
	OperatorUsername string            `json:"operatorUsername" description:"操作者用户名"`
	Action           string            `json:"action" description:"操作"`
	ResourceType     string            `json:"resourceType" description:"资源类型"`
	ResourceID       string            `json:"resourceId" description:"资源ID"`
	Method           string            `json:"method" description:"HTTP方法"`
	RoutePath        string            `json:"routePath" description:"匹配的路由模板"`
	Summary          map[string]string `json:"summary" description:"不含敏感值的变更摘要"`
	Result           string            `json:"result" description:"执行结果"`
	StatusCode       int               `json:"statusCode" description:"HTTP状态码"`
	RequestID        string            `json:"requestId" description:"请求ID"`
	RemoteIP         string            `json:"remoteIp" description:"客户端直连IP"`
	CreatedAt        time.Time         `json:"createdAt" description:"操作时间"`
}

type AuditLogList struct {
	Data  []AuditLogDetail `json:"data"`
	Total int64            `json:"total"`
}

type ModelDiagnosticInput struct {
	ModelID         string `json:"modelId" validate:"required" description:"统一模型ID"`
	Prompt          string `json:"prompt" validate:"required,max=16000" description:"演练提示词；不持久化"`
	MaxOutputTokens int64  `json:"maxOutputTokens" validate:"gte=1,lte=32768" description:"最大输出Token"`
}

type ModelDiagnosticAttempt struct {
	Sequence      int    `json:"sequence" description:"尝试序号"`
	RouteID       string `json:"routeId" description:"路由ID"`
	ChannelID     string `json:"channelId" description:"渠道ID"`
	ChannelName   string `json:"channelName" description:"渠道名称"`
	ProviderName  string `json:"providerName" description:"供应商名称"`
	UpstreamModel string `json:"upstreamModel" description:"上游模型"`
	HTTPStatus    int    `json:"httpStatus" description:"HTTP状态"`
	DurationMS    int64  `json:"durationMs" description:"本次尝试耗时毫秒"`
	ErrorCode     string `json:"errorCode" description:"错误码"`
	ErrorSummary  string `json:"errorSummary" description:"脱敏错误摘要"`
}

type ModelDiagnosticResult struct {
	Success          bool                     `json:"success" description:"演练是否成功"`
	RequestID        string                   `json:"requestId" description:"演练请求ID"`
	ModelID          string                   `json:"modelId" description:"统一模型ID"`
	ModelName        string                   `json:"modelName" description:"统一模型名称"`
	OutputText       string                   `json:"outputText" description:"模型输出；不持久化"`
	TTFTMS           int64                    `json:"ttftMs" description:"首Token耗时毫秒"`
	DurationMS       int64                    `json:"durationMs" description:"总耗时毫秒"`
	PromptTokens     int64                    `json:"promptTokens" description:"输入Token"`
	CompletionTokens int64                    `json:"completionTokens" description:"输出Token"`
	TotalTokens      int64                    `json:"totalTokens" description:"总Token"`
	Estimated        bool                     `json:"estimated" description:"Token是否估算"`
	FinalChannelID   string                   `json:"finalChannelId" description:"最终渠道ID"`
	FinalChannelName string                   `json:"finalChannelName" description:"最终渠道名称"`
	Message          string                   `json:"message" description:"结果摘要"`
	Attempts         []ModelDiagnosticAttempt `json:"attempts" description:"本次路由尝试链"`
}
