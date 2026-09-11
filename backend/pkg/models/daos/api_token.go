package daos

import (
	"time"

	"github.com/efucloud/token-router/pkg/models"
)

type APIToken struct {
	GatewayRecord
	AccountID        string     `gorm:"column:account_id;type:varchar(50);not null" json:"accountId"`
	Name             string     `gorm:"column:name;type:varchar(255);not null" json:"name"`
	KeyHash          string     `gorm:"column:key_hash;type:char(64);not null" json:"-"`
	KeyPrefix        string     `gorm:"column:key_prefix;type:varchar(20);not null" json:"keyPrefix"`
	Status           string     `gorm:"column:status;type:varchar(32);default:active;not null" json:"status"`
	ExpiresAt        *time.Time `gorm:"column:expires_at" json:"expiresAt,omitempty"`
	AllowedModels    string     `gorm:"column:allowed_models;type:text;not null" json:"allowedModels"`
	TokenLimit       int64      `gorm:"column:token_limit;default:0;not null" json:"tokenLimit"`
	RequestLimit     int64      `gorm:"column:request_limit;default:0;not null" json:"requestLimit"`
	UsedTokens       int64      `gorm:"column:used_tokens;default:0;not null" json:"usedTokens"`
	UsedRequests     int64      `gorm:"column:used_requests;default:0;not null" json:"usedRequests"`
	ReservedTokens   int64      `gorm:"column:reserved_tokens;default:0;not null" json:"-"`
	ReservedRequests int64      `gorm:"column:reserved_requests;default:0;not null" json:"-"`
	IPAllowlist      string     `gorm:"column:ip_allowlist;type:text;not null" json:"ipAllowlist"`
	RPM              int        `gorm:"column:rpm;default:0;not null" json:"rpm"`
	TPM              int64      `gorm:"column:tpm;default:0;not null" json:"tpm"`
	MaxConcurrency   int        `gorm:"column:max_concurrency;default:0;not null" json:"maxConcurrency"`
}

func (*APIToken) TableName() string { return models.APITokenTableName }

func (*APIToken) Indexes() map[string][]string {
	return map[string][]string{
		"idx_api_token_account": {"account_id", "status"},
		"idx_api_token_prefix":  {"key_prefix"},
	}
}

func (*APIToken) UniqueIndexes() map[string][]string {
	return map[string][]string{"uniq_idx_api_token_hash": {"key_hash"}}
}
