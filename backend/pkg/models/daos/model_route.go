package daos

import "github.com/efucloud/token-router/pkg/models"

type ModelRoute struct {
	GatewayRecord
	ModelID       string `gorm:"column:model_id;type:varchar(50);not null" json:"modelId"`
	ChannelID     string `gorm:"column:channel_id;type:varchar(50);not null" json:"channelId"`
	UpstreamModel string `gorm:"column:upstream_model;type:varchar(255);not null" json:"upstreamModel"`
	Priority      int    `gorm:"column:priority;default:0;not null" json:"priority"`
	Weight        int    `gorm:"column:weight;default:1;not null" json:"weight"`
	Status        string `gorm:"column:status;type:varchar(32);default:enabled;not null" json:"status"`
	Version       int64  `gorm:"column:version;default:1;not null" json:"version"`
}

func (*ModelRoute) TableName() string { return models.ModelRouteTableName }

func (*ModelRoute) Indexes() map[string][]string {
	return map[string][]string{
		"idx_model_route_model":   {"model_id", "status", "priority"},
		"idx_model_route_channel": {"channel_id"},
	}
}

func (*ModelRoute) UniqueIndexes() map[string][]string {
	return map[string][]string{"uniq_idx_model_route_mapping": {"model_id", "channel_id", "upstream_model"}}
}
