package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/efucloud/token-router/pkg/config"
)

func TestOIDCRoleForEmail(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		adminEmails []string
		want        string
	}{
		{
			name:        "configured administrator",
			email:       "admin@example.com",
			adminEmails: []string{"admin@example.com"},
			want:        "admin",
		},
		{
			name:        "email comparison ignores case and surrounding spaces",
			email:       " Admin@Example.com ",
			adminEmails: []string{"other@example.com", " admin@example.com "},
			want:        "admin",
		},
		{
			name:        "regular user",
			email:       "user@example.com",
			adminEmails: []string{"admin@example.com"},
			want:        "none",
		},
		{
			name:        "empty administrator list",
			email:       "admin@example.com",
			adminEmails: nil,
			want:        "none",
		},
		{
			name:        "empty email is never an administrator",
			email:       " ",
			adminEmails: []string{" "},
			want:        "none",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := oidcRoleForEmail(tt.email, tt.adminEmails); got != tt.want {
				t.Fatalf("oidcRoleForEmail() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOIDCAccountClaimsSupportsStandardUserinfoFields(t *testing.T) {
	create, err := (oidcAccountClaims{
		Subject:           "provider-subject",
		PreferredUsername: "external-user",
		Name:              "External User",
		Email:             "admin@example.com",
		PhoneNumber:       "+86-1234",
		Locale:            "en-GB",
		Picture:           "https://identity.example/avatar.png",
	}).accountCreate("verified-account-id", []string{" ADMIN@example.com "})
	if err != nil {
		t.Fatalf("convert userinfo claims: %v", err)
	}
	if create.ID != "verified-account-id" || create.Username != "external-user" || create.Nickname != "External User" {
		t.Fatalf("unexpected identity fields: %#v", create)
	}
	if create.Role != "admin" || create.Phone != "+86-1234" || create.Language != "en-US" || create.Avatar == "" {
		t.Fatalf("unexpected profile fields: %#v", create)
	}
}

func TestProvisionAccountFromTokenUsesOIDCUserinfo(t *testing.T) {
	db := dashboardTestDatabase(t)
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/.well-known/openid-configuration":
			writer.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(writer).Encode(map[string]interface{}{
				"issuer": server.URL, "authorization_endpoint": server.URL + "/authorize",
				"token_endpoint": server.URL + "/token", "jwks_uri": server.URL + "/keys",
				"userinfo_endpoint": server.URL + "/userinfo", "response_types_supported": []string{"code"},
				"subject_types_supported": []string{"public"}, "id_token_signing_alg_values_supported": []string{"RS256"},
			})
		case "/userinfo":
			if request.Header.Get("Authorization") != "Bearer external-token" {
				http.Error(writer, "missing bearer token", http.StatusUnauthorized)
				return
			}
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"sub":"provider-subject","preferred_username":"external-user","name":"External User","email":"user@example.com","locale":"zh-CN"}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	t.Cleanup(server.Close)

	provider, err := oidc.NewProvider(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("create test OIDC provider: %v", err)
	}
	previousProvider := config.AuthProvider
	config.AuthProvider = provider
	t.Cleanup(func() { config.AuthProvider = previousProvider })

	service := OAuthService{}
	account, errorData := service.ProvisionAccountFromToken(context.Background(), "external-token", "verified-account-id")
	if errorData.IsNotNil() {
		t.Fatalf("provision account: %v", errorData.Err)
	}
	if account.ID != "verified-account-id" || account.Username != "external-user" || !account.Enable {
		t.Fatalf("unexpected provisioned account: %#v", account)
	}
	var count int64
	if err = db.Table("account").Where("id = ?", account.ID).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("account was not persisted exactly once: count=%d err=%v", count, err)
	}

	if err = db.Table("account").Where("id = ?", account.ID).Update("enable", false).Error; err != nil {
		t.Fatalf("disable account: %v", err)
	}
	account, errorData = service.ProvisionAccountFromToken(context.Background(), "external-token", "verified-account-id")
	if errorData.IsNotNil() || account.Enable {
		t.Fatalf("existing disabled account was changed: account=%#v err=%v", account, errorData.Err)
	}
}
