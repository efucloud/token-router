package filters

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/efucloud/token-router/pkg/config"
	"github.com/efucloud/token-router/pkg/models/daos"
	"github.com/efucloud/token-router/pkg/models/dtos"
	restful "github.com/emicklei/go-restful/v3"
	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestAccountIDFromClaimsSupportsSharedOIDCTokens(t *testing.T) {
	tests := []struct {
		name   string
		claims dtos.UserClaims
		want   string
	}{
		{name: "eauth identity", claims: dtos.UserClaims{EAuthId: "eauth-user", ID: "id-user", RegisteredClaims: jwt.RegisteredClaims{Subject: "subject-user"}}, want: "eauth-user"},
		{name: "provider id fallback", claims: dtos.UserClaims{ID: "id-user", RegisteredClaims: jwt.RegisteredClaims{Subject: "subject-user"}}, want: "id-user"},
		{name: "standard subject fallback", claims: dtos.UserClaims{RegisteredClaims: jwt.RegisteredClaims{Subject: "subject-user"}}, want: "subject-user"},
		{name: "missing identity", claims: dtos.UserClaims{}, want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := accountIDFromClaims(test.claims); got != test.want {
				t.Fatalf("expected %q, got %q", test.want, got)
			}
		})
	}
}

func TestAuthAutoCreatesMissingAccountFromOIDCUserinfo(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate signing key: %v", err)
	}
	const keyID = "test-key"
	var accessToken string
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(writer).Encode(map[string]interface{}{
				"issuer": server.URL, "authorization_endpoint": server.URL + "/authorize",
				"token_endpoint": server.URL + "/token", "jwks_uri": server.URL + "/keys",
				"userinfo_endpoint": server.URL + "/userinfo", "response_types_supported": []string{"code"},
				"subject_types_supported": []string{"public"}, "id_token_signing_alg_values_supported": []string{"RS256"},
			})
		case "/keys":
			_ = json.NewEncoder(writer).Encode(map[string]interface{}{"keys": []map[string]string{{
				"kty": "RSA", "kid": keyID, "use": "sig", "alg": "RS256",
				"n": base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes()), "e": "AQAB",
			}}})
		case "/userinfo":
			if request.Header.Get("Authorization") != "Bearer "+accessToken {
				http.Error(writer, `{"error":"invalid_token"}`, http.StatusUnauthorized)
				return
			}
			_, _ = writer.Write([]byte(`{"sub":"oidc-subject","preferred_username":"api-user","name":"API User","email":"api-user@example.com"}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	t.Cleanup(server.Close)

	provider, err := oidc.NewProvider(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("create test OIDC provider: %v", err)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss": server.URL, "sub": "oidc-subject", "id": "auto-created-user", "aud": "token-router",
		"iat": time.Now().Add(-time.Minute).Unix(), "exp": time.Now().Add(time.Hour).Unix(),
	})
	token.Header["kid"] = keyID
	accessToken, err = token.SignedString(key)
	if err != nil {
		t.Fatalf("sign access token: %v", err)
	}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open auth database: %v", err)
	}
	if err = db.AutoMigrate(&daos.Account{}); err != nil {
		t.Fatalf("migrate auth database: %v", err)
	}
	previousDB, previousProvider := config.DBConnect, config.AuthProvider
	previousVerifier, previousLogger := config.SystemVerifier, config.Logger
	config.DBConnect = db
	config.AuthProvider = provider
	config.SystemVerifier = provider.Verifier(&oidc.Config{ClientID: "token-router"})
	config.Logger = zap.NewNop().Sugar()
	t.Cleanup(func() {
		config.DBConnect, config.AuthProvider = previousDB, previousProvider
		config.SystemVerifier, config.Logger = previousVerifier, previousLogger
	})

	container := restful.NewContainer()
	webService := new(restful.WebService)
	webService.Route(webService.GET("/protected").Filter(Auth).To(func(_ *restful.Request, response *restful.Response) {
		response.WriteHeader(http.StatusOK)
	}))
	container.Add(webService)
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set(config.AuthHeader, "Bearer "+accessToken)
	recorder := httptest.NewRecorder()
	container.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("first token request was rejected: status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	var account daos.Account
	if err = db.First(&account, "id = ?", "auto-created-user").Error; err != nil {
		t.Fatalf("read auto-created account: %v", err)
	}
	if account.Username != "api-user" || account.Nickname != "API User" || !account.Enable {
		t.Fatalf("unexpected auto-created account: %#v", account)
	}
}
