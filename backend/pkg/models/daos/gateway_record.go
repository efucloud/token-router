package daos

import "time"

// GatewayRecord contains fields shared by gateway control-plane entities.
type GatewayRecord struct {
	ID        string    `gorm:"primarykey;column:id;type:varchar(50)" json:"id"`
	CreatedAt time.Time `gorm:"autoCreateTime;column:created_at;<-:create" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime;column:updated_at" json:"updatedAt"`
}
