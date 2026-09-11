package filters

import (
	"testing"

	"github.com/efucloud/token-router/pkg/models/dtos"
	"github.com/golang-jwt/jwt/v5"
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
