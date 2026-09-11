package daos

import "github.com/efucloud/token-router/pkg/models"

type RouteAttempt struct {
	GatewayRecord
	RequestID    string `gorm:"column:request_id;type:varchar(64);not null" json:"requestId"`
	RouteID      string `gorm:"column:route_id;type:varchar(50);not null" json:"routeId"`
	ChannelID    string `gorm:"column:channel_id;type:varchar(50);not null" json:"channelId"`
	Sequence     int    `gorm:"column:sequence;not null" json:"sequence"`
	HTTPStatus   int    `gorm:"column:http_status;default:0;not null" json:"httpStatus"`
	DurationMS   int64  `gorm:"column:duration_ms;default:0;not null" json:"durationMs"`
	ErrorCode    string `gorm:"column:error_code;type:varchar(64)" json:"errorCode,omitempty"`
	ErrorSummary string `gorm:"column:error_summary;type:varchar(500)" json:"errorSummary,omitempty"`
}

func (*RouteAttempt) TableName() string { return models.RouteAttemptTableName }

func (*RouteAttempt) Indexes() map[string][]string {
	return map[string][]string{
		"idx_route_attempt_request": {"request_id"},
		"idx_route_attempt_channel": {"channel_id", "created_at"},
	}
}

func (*RouteAttempt) UniqueIndexes() map[string][]string {
	return map[string][]string{"uniq_idx_route_attempt_sequence": {"request_id", "sequence"}}
}
