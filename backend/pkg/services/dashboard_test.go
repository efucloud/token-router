package services

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/efucloud/token-router/pkg/config"
	"github.com/efucloud/token-router/pkg/models/daos"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func dashboardTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err = db.AutoMigrate(&daos.Account{}, &daos.APIToken{}, &daos.AIModel{}, &daos.Provider{}, &daos.Channel{}, &daos.ModelRoute{}, &daos.UsageLog{}, &daos.RouteAttempt{}, &daos.APIRateLimitBucket{}, &daos.ConcurrencyLease{}, &daos.AuditLog{}, &daos.ChatConversation{}, &daos.ChatMessage{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	if err = db.Exec("CREATE UNIQUE INDEX uniq_test_rate_bucket ON api_rate_limit_bucket(api_token_id, window_start)").Error; err != nil {
		t.Fatalf("create rate bucket test index: %v", err)
	}
	previous := config.DBConnect
	config.DBConnect = db
	t.Cleanup(func() { config.DBConnect = previous })
	return db
}

func TestDashboardScopesUserAndAdminData(t *testing.T) {
	db := dashboardTestDatabase(t)
	now := time.Now()
	accounts := []daos.Account{
		{ID: "u1", Username: "alice", Nickname: "Alice", Enable: true, TokenLimit: 1000},
		{ID: "u2", Username: "bob", Nickname: "Bob", Enable: true, TokenLimit: 1000},
	}
	if err := db.Create(&accounts).Error; err != nil {
		t.Fatalf("create accounts: %v", err)
	}
	tokens := []daos.APIToken{
		{GatewayRecord: daos.GatewayRecord{ID: "t1"}, AccountID: "u1", Name: "alice-token", KeyHash: "hash1", KeyPrefix: "tr_alice", Status: "active", AllowedModels: "[]", IPAllowlist: "[]"},
		{GatewayRecord: daos.GatewayRecord{ID: "t2"}, AccountID: "u2", Name: "bob-token", KeyHash: "hash2", KeyPrefix: "tr_bob", Status: "active", AllowedModels: "[]", IPAllowlist: "[]"},
	}
	if err := db.Create(&tokens).Error; err != nil {
		t.Fatalf("create tokens: %v", err)
	}
	logs := []daos.UsageLog{
		{GatewayRecord: daos.GatewayRecord{ID: "l1", CreatedAt: now.Add(-time.Hour)}, RequestID: "r1", AccountID: "u1", APITokenID: "t1", TokenPrefix: "tr_alice", ModelID: "m1", ModelName: "model", Endpoint: "chat/completions", Status: "succeeded", TotalTokens: 10, DurationMS: 100},
		{GatewayRecord: daos.GatewayRecord{ID: "l2", CreatedAt: now.Add(-time.Hour)}, RequestID: "r2", AccountID: "u2", APITokenID: "t2", TokenPrefix: "tr_bob", ModelID: "m1", ModelName: "model", Endpoint: "chat/completions", Status: "succeeded", TotalTokens: 20, DurationMS: 200},
	}
	if err := db.Create(&logs).Error; err != nil {
		t.Fatalf("create usage logs: %v", err)
	}

	ctx := context.WithValue(context.Background(), config.RequestUserId, "u1")
	my, errorData := (DashboardService{}).GetMyDashboard(ctx, "24h")
	if errorData.IsNotNil() {
		t.Fatalf("get user dashboard: %v", errorData.Err)
	}
	if my.Summary.TotalRequests != 1 || my.Summary.TotalTokens != 10 {
		t.Fatalf("user dashboard leaked another user's data: %#v", my.Summary)
	}
	admin, errorData := (DashboardService{}).GetAdminDashboard(ctx, "24h")
	if errorData.IsNotNil() {
		t.Fatalf("get admin dashboard: %v", errorData.Err)
	}
	if admin.Summary.TotalRequests != 2 || admin.Summary.TotalTokens != 30 || admin.Principals.ActiveUsers != 2 {
		t.Fatalf("unexpected admin dashboard summary: %#v", admin)
	}

	bobContext := context.WithValue(context.Background(), config.RequestUserId, "u2")
	bobDashboard, errorData := (DashboardService{}).GetMyDashboard(bobContext, "24h")
	if errorData.IsNotNil() {
		t.Fatalf("get bob dashboard: %v", errorData.Err)
	}
	adminView, errorData := (DashboardService{}).GetUserDashboard(ctx, "u2", "24h")
	if errorData.IsNotNil() {
		t.Fatalf("admin get bob dashboard: %v", errorData.Err)
	}
	if !reflect.DeepEqual(adminView.Summary, bobDashboard.Summary) ||
		!reflect.DeepEqual(adminView.Quota, bobDashboard.Quota) ||
		!reflect.DeepEqual(adminView.TokenStatus, bobDashboard.TokenStatus) ||
		!reflect.DeepEqual(adminView.ModelUsage, bobDashboard.ModelUsage) ||
		!reflect.DeepEqual(adminView.RecentRequests, bobDashboard.RecentRequests) {
		t.Fatalf("admin user dashboard differs from the user's own dashboard: admin=%#v user=%#v", adminView, bobDashboard)
	}
}
