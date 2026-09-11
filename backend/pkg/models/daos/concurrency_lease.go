package daos

import (
	"time"

	"github.com/efucloud/token-router/pkg/models"
)

type ConcurrencyLease struct {
	GatewayRecord
	APITokenID string    `gorm:"column:api_token_id;type:varchar(50);not null" json:"apiTokenId"`
	ExpiresAt  time.Time `gorm:"column:expires_at;not null" json:"expiresAt"`
}

func (*ConcurrencyLease) TableName() string { return models.ConcurrencyLeaseTableName }

func (*ConcurrencyLease) Indexes() map[string][]string {
	return map[string][]string{"idx_concurrency_lease_token_expiry": {"api_token_id", "expires_at"}}
}

func (*ConcurrencyLease) UniqueIndexes() map[string][]string { return map[string][]string{} }
