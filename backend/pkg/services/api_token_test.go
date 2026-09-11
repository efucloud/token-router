package services

import (
	"context"
	"strings"
	"testing"

	"github.com/efucloud/token-router/pkg/config"
	"github.com/efucloud/token-router/pkg/models/daos"
	"github.com/efucloud/token-router/pkg/models/dtos"
)

func TestAPITokenServiceScopesTokensToCurrentUser(t *testing.T) {
	db := dashboardTestDatabase(t)
	accounts := []daos.Account{
		{ID: "u1", Username: "alice", Enable: true},
		{ID: "u2", Username: "bob", Enable: true},
	}
	if err := db.Create(&accounts).Error; err != nil {
		t.Fatalf("create accounts: %v", err)
	}
	service := APITokenService{}
	alice := context.WithValue(context.Background(), config.RequestUserId, "u1")
	created, errorData := service.Create(alice, dtos.APITokenCreate{Name: "alice-token", AllowedModels: []string{"model"}, TokenLimit: 100, RequestLimit: 10})
	if errorData.IsNotNil() {
		t.Fatalf("create token: %v", errorData.Err)
	}
	if !strings.HasPrefix(created.Token, "tr_") || created.KeyPrefix == "" {
		t.Fatalf("created token did not contain one-time plaintext and prefix: %#v", created)
	}
	listed, errorData := service.List(alice)
	if errorData.IsNotNil() || listed.Total != 1 || len(listed.Data) != 1 {
		t.Fatalf("list Alice tokens: %#v, %v", listed, errorData.Err)
	}
	bob := context.WithValue(context.Background(), config.RequestUserId, "u2")
	listed, errorData = service.List(bob)
	if errorData.IsNotNil() || listed.Total != 0 {
		t.Fatalf("Bob must not see Alice token: %#v, %v", listed, errorData.Err)
	}
	if errorData = service.Delete(bob, created.ID); errorData.IsNil() {
		t.Fatal("Bob must not delete Alice token")
	}
}
