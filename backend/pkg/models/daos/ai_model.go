package daos

import "github.com/efucloud/token-router/pkg/models"

type AIModel struct {
	GatewayRecord
	Name            string `gorm:"column:name;type:varchar(255);not null" json:"name"`
	DisplayName     string `gorm:"column:display_name;type:varchar(255);not null" json:"displayName"`
	Description     string `gorm:"column:description;type:varchar(1000);not null" json:"description"`
	Modality        string `gorm:"column:modality;type:varchar(32);not null" json:"modality"`
	ContextWindow   int64  `gorm:"column:context_window;default:0;not null" json:"contextWindow"`
	MaxOutputTokens int64  `gorm:"column:max_output_tokens;default:0;not null" json:"maxOutputTokens"`
	Capabilities    string `gorm:"column:capabilities;type:text;not null" json:"capabilities"`
	Status          string `gorm:"column:status;type:varchar(32);default:active;not null" json:"status"`
	Version         int64  `gorm:"column:version;default:1;not null" json:"version"`
}

func (*AIModel) TableName() string { return models.AIModelTableName }

func (*AIModel) Indexes() map[string][]string {
	return map[string][]string{"idx_ai_model_status": {"status"}}
}

func (*AIModel) UniqueIndexes() map[string][]string {
	return map[string][]string{"uniq_idx_ai_model_name": {"name"}}
}
