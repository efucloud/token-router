package daos

import (
	"time"

	"github.com/efucloud/token-router/pkg/models"
)

type APIRateLimitBucket struct {
	GatewayRecord
	APITokenID     string    `gorm:"column:api_token_id;type:varchar(50);not null" json:"apiTokenId"`
	WindowStart    time.Time `gorm:"column:window_start;not null" json:"windowStart"`
	UsedRequests   int64     `gorm:"column:used_requests;default:0;not null" json:"usedRequests"`
	UsedTokens     int64     `gorm:"column:used_tokens;default:0;not null" json:"usedTokens"`
	ReservedTokens int64     `gorm:"column:reserved_tokens;default:0;not null" json:"reservedTokens"`
}

func (*APIRateLimitBucket) TableName() string { return models.APIRateLimitBucketTableName }

func (*APIRateLimitBucket) Indexes() map[string][]string {
	return map[string][]string{"idx_api_rate_bucket_window": {"window_start"}}
}

func (*APIRateLimitBucket) UniqueIndexes() map[string][]string {
	return map[string][]string{"uniq_idx_api_rate_bucket_token_window": {"api_token_id", "window_start"}}
}
