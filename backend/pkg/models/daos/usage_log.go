package daos

import "github.com/efucloud/token-router/pkg/models"

type UsageLog struct {
	GatewayRecord
	RequestID        string `gorm:"column:request_id;type:varchar(64);not null" json:"requestId"`
	AccountID        string `gorm:"column:account_id;type:varchar(50);not null" json:"accountId"`
	APITokenID       string `gorm:"column:api_token_id;type:varchar(50);not null" json:"apiTokenId"`
	TokenPrefix      string `gorm:"column:token_prefix;type:varchar(20);not null" json:"tokenPrefix"`
	ModelID          string `gorm:"column:model_id;type:varchar(50);not null" json:"modelId"`
	ModelName        string `gorm:"column:model_name;type:varchar(255);not null" json:"modelName"`
	ChannelID        string `gorm:"column:channel_id;type:varchar(50)" json:"channelId,omitempty"`
	Endpoint         string `gorm:"column:endpoint;type:varchar(64);not null" json:"endpoint"`
	Streaming        bool   `gorm:"column:streaming;default:false;not null" json:"streaming"`
	PromptTokens     int64  `gorm:"column:prompt_tokens;default:0;not null" json:"promptTokens"`
	CompletionTokens int64  `gorm:"column:completion_tokens;default:0;not null" json:"completionTokens"`
	TotalTokens      int64  `gorm:"column:total_tokens;default:0;not null" json:"totalTokens"`
	ReservedTokens   int64  `gorm:"column:reserved_tokens;default:0;not null" json:"reservedTokens"`
	Estimated        bool   `gorm:"column:estimated;default:false;not null" json:"estimated"`
	Status           string `gorm:"column:status;type:varchar(32);not null" json:"status"`
	HTTPStatus       int    `gorm:"column:http_status;default:0;not null" json:"httpStatus"`
	DurationMS       int64  `gorm:"column:duration_ms;default:0;not null" json:"durationMs"`
	ErrorCode        string `gorm:"column:error_code;type:varchar(64)" json:"errorCode,omitempty"`
	ErrorSummary     string `gorm:"column:error_summary;type:varchar(500)" json:"errorSummary,omitempty"`
}

func (*UsageLog) TableName() string { return models.UsageLogTableName }

func (*UsageLog) Indexes() map[string][]string {
	return map[string][]string{
		"idx_usage_account_time": {"account_id", "created_at"},
		"idx_usage_model_time":   {"model_id", "created_at"},
		"idx_usage_channel_time": {"channel_id", "created_at"},
		"idx_usage_status_time":  {"status", "created_at"},
	}
}

func (*UsageLog) UniqueIndexes() map[string][]string {
	return map[string][]string{"uniq_idx_usage_request": {"request_id"}}
}
