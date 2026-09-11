package daos

import (
	"time"

	"github.com/efucloud/token-router/pkg/models"
)

type Channel struct {
	GatewayRecord
	ProviderID          string     `gorm:"column:provider_id;type:varchar(50);not null" json:"providerId"`
	Name                string     `gorm:"column:name;type:varchar(255);not null" json:"name"`
	BaseURL             string     `gorm:"column:base_url;type:varchar(1000);not null" json:"baseUrl"`
	EncryptedAPIKey     string     `gorm:"column:encrypted_api_key;type:text;not null" json:"-"`
	Priority            int        `gorm:"column:priority;default:0;not null" json:"priority"`
	Weight              int        `gorm:"column:weight;default:1;not null" json:"weight"`
	Status              string     `gorm:"column:status;type:varchar(32);default:enabled;not null" json:"status"`
	TimeoutSeconds      int        `gorm:"column:timeout_seconds;default:0;not null" json:"timeoutSeconds"`
	MaxConcurrency      int        `gorm:"column:max_concurrency;default:0;not null" json:"maxConcurrency"`
	HealthStatus        string     `gorm:"column:health_status;type:varchar(32);default:unknown;not null" json:"healthStatus"`
	CooldownUntil       *time.Time `gorm:"column:cooldown_until" json:"cooldownUntil,omitempty"`
	ConsecutiveFailures int        `gorm:"column:consecutive_failures;default:0;not null" json:"consecutiveFailures"`
	LastCheckedAt       *time.Time `gorm:"column:last_checked_at" json:"lastCheckedAt,omitempty"`
	LastLatencyMS       int64      `gorm:"column:last_latency_ms;default:0;not null" json:"lastLatencyMs"`
	LastError           string     `gorm:"column:last_error;type:varchar(500);not null" json:"lastError"`
	Version             int64      `gorm:"column:version;default:1;not null" json:"version"`
}

func (*Channel) TableName() string { return models.ChannelTableName }

func (*Channel) Indexes() map[string][]string {
	return map[string][]string{
		"idx_channel_provider": {"provider_id"},
		"idx_channel_status":   {"status", "health_status", "cooldown_until"},
	}
}

func (*Channel) UniqueIndexes() map[string][]string {
	return map[string][]string{"uniq_idx_channel_provider_name": {"provider_id", "name"}}
}
