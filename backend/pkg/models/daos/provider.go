package daos

import "github.com/efucloud/token-router/pkg/models"

type Provider struct {
	GatewayRecord
	Name    string `gorm:"column:name;type:varchar(255);not null" json:"name"`
	Type    string `gorm:"column:type;type:varchar(64);not null" json:"type"`
	Status  string `gorm:"column:status;type:varchar(32);default:enabled;not null" json:"status"`
	Config  string `gorm:"column:config;type:text;not null" json:"config"`
	Version int64  `gorm:"column:version;default:1;not null" json:"version"`
}

func (*Provider) TableName() string { return models.ProviderTableName }

func (*Provider) Indexes() map[string][]string {
	return map[string][]string{"idx_provider_status": {"status"}}
}

func (*Provider) UniqueIndexes() map[string][]string {
	return map[string][]string{"uniq_idx_provider_name": {"name"}}
}
