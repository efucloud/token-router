package utils

import (
	"strings"
	"testing"
)

func TestGenerateAPIToken(t *testing.T) {
	first, firstHash, firstPrefix, err := GenerateAPIToken()
	if err != nil {
		t.Fatalf("generate API key: %v", err)
	}
	second, secondHash, _, err := GenerateAPIToken()
	if err != nil {
		t.Fatalf("generate second API key: %v", err)
	}
	if !strings.HasPrefix(first, APITokenPrefix) || first == second || firstHash == secondHash {
		t.Fatal("generated tokens must have the tr_ prefix and be unique")
	}
	if !strings.HasPrefix(first, firstPrefix) || len(firstPrefix) != apiTokenDisplayChars {
		t.Fatalf("unexpected display prefix %q", firstPrefix)
	}
	if !APITokenMatches(first, firstHash) || APITokenMatches(second, firstHash) {
		t.Fatal("token digest matching returned an incorrect result")
	}
}

func TestHashAPITokenRejectsInvalidFormat(t *testing.T) {
	for _, token := range []string{"", "tr_", "sk_not-a-token"} {
		if _, err := HashAPIToken(token); err == nil {
			t.Fatalf("expected %q to be rejected", token)
		}
	}
}
