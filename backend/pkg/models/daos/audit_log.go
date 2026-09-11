package daos

import "github.com/efucloud/token-router/pkg/models"

type AuditLog struct {
	GatewayRecord
	OperatorID   string `gorm:"column:operator_id;type:varchar(50);not null" json:"operatorId"`
	Action       string `gorm:"column:action;type:varchar(50);not null" json:"action"`
	ResourceType string `gorm:"column:resource_type;type:varchar(100);not null" json:"resourceType"`
	ResourceID   string `gorm:"column:resource_id;type:varchar(100);not null" json:"resourceId"`
	Method       string `gorm:"column:method;type:varchar(10);not null" json:"method"`
	RoutePath    string `gorm:"column:route_path;type:varchar(500);not null" json:"routePath"`
	Summary      string `gorm:"column:summary;type:text;not null" json:"summary"`
	Result       string `gorm:"column:result;type:varchar(20);not null" json:"result"`
	StatusCode   int    `gorm:"column:status_code;not null" json:"statusCode"`
	RequestID    string `gorm:"column:request_id;type:varchar(64);not null" json:"requestId"`
	RemoteIP     string `gorm:"column:remote_ip;type:varchar(100);not null" json:"remoteIp"`
}

func (*AuditLog) TableName() string { return models.AuditLogTableName }

func (*AuditLog) Indexes() map[string][]string {
	return map[string][]string{
		"idx_audit_log_created":  {"created_at"},
		"idx_audit_log_operator": {"operator_id", "created_at"},
		"idx_audit_log_resource": {"resource_type", "resource_id", "created_at"},
		"idx_audit_log_request":  {"request_id"},
	}
}

func (*AuditLog) UniqueIndexes() map[string][]string { return map[string][]string{} }
