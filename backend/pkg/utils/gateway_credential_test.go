package utils

import (
	"encoding/base64"
	"strings"
	"testing"
)

func gatewayTestKey(fill byte) string {
	return base64.StdEncoding.EncodeToString([]byte(strings.Repeat(string(fill), 32)))
}

func TestGatewayCredentialRoundTrip(t *testing.T) {
	key := gatewayTestKey('a')
	first, err := EncryptGatewayCredential("sk-secret", key)
	if err != nil {
		t.Fatalf("encrypt credential: %v", err)
	}
	second, err := EncryptGatewayCredential("sk-secret", key)
	if err != nil {
		t.Fatalf("encrypt credential again: %v", err)
	}
	if first == second {
		t.Fatal("ciphertexts must differ because each encryption needs a new nonce")
	}
	plaintext, err := DecryptGatewayCredential(first, key)
	if err != nil {
		t.Fatalf("decrypt credential: %v", err)
	}
	if plaintext != "sk-secret" {
		t.Fatalf("unexpected plaintext %q", plaintext)
	}
}

func TestGatewayCredentialRejectsInvalidKeyAndTampering(t *testing.T) {
	if _, err := EncryptGatewayCredential("secret", base64.StdEncoding.EncodeToString([]byte("short"))); err == nil {
		t.Fatal("expected invalid key length to fail")
	}
	key := gatewayTestKey('a')
	encrypted, err := EncryptGatewayCredential("secret", key)
	if err != nil {
		t.Fatalf("encrypt credential: %v", err)
	}
	last := encrypted[len(encrypted)-1]
	replacement := byte('A')
	if last == replacement {
		replacement = 'B'
	}
	tampered := encrypted[:len(encrypted)-1] + string(replacement)
	if _, err = DecryptGatewayCredential(tampered, key); err == nil {
		t.Fatal("expected tampered ciphertext to fail")
	}
	if _, err = DecryptGatewayCredential(encrypted, gatewayTestKey('b')); err == nil {
		t.Fatal("expected wrong key to fail")
	}
}
