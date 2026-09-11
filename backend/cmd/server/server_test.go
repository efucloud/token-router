package server

import (
	"testing"

	"github.com/efucloud/token-router/pkg/config"
)

func TestOIDCVerifierConfigControlsAudienceCheck(t *testing.T) {
	strict := oidcVerifierConfig(config.OidcConfig{ClientId: "token-router"})
	if strict.ClientID != "token-router" || strict.SkipClientIDCheck {
		t.Fatalf("unexpected strict verifier config: %#v", strict)
	}

	shared := oidcVerifierConfig(config.OidcConfig{ClientId: "token-router", SkipClientIDCheck: true})
	if shared.ClientID != "token-router" || !shared.SkipClientIDCheck {
		t.Fatalf("unexpected shared verifier config: %#v", shared)
	}
}
