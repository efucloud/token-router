package utils

import (
	"testing"
)

func TestGatewayCredentialRoundTrip(t *testing.T) {
	key, err := decodeGatewayMasterKey()
	if err != nil || len(key) != 32 {
		t.Fatalf("invalid built-in master key: length=%d err=%v", len(key), err)
	}
	first, err := EncryptGatewayCredential("sk-secret")
	if err != nil {
		t.Fatalf("encrypt credential: %v", err)
	}
	second, err := EncryptGatewayCredential("sk-secret")
	if err != nil {
		t.Fatalf("encrypt credential again: %v", err)
	}
	if first == second {
		t.Fatal("ciphertexts must differ because each encryption needs a new nonce")
	}
	plaintext, err := DecryptGatewayCredential(first)
	if err != nil {
		t.Fatalf("decrypt credential: %v", err)
	}
	if plaintext != "sk-secret" {
		t.Fatalf("unexpected plaintext %q", plaintext)
	}
}

func TestGatewayCredentialRejectsTampering(t *testing.T) {
	encrypted, err := EncryptGatewayCredential("secret")
	if err != nil {
		t.Fatalf("encrypt credential: %v", err)
	}
	tamperedIndex := len(gatewayCredentialVersion) + 1
	current := encrypted[tamperedIndex]
	replacement := byte('A')
	if current == replacement {
		replacement = 'B'
	}
	tampered := encrypted[:tamperedIndex] + string(replacement) + encrypted[tamperedIndex+1:]
	if _, err = DecryptGatewayCredential(tampered); err == nil {
		t.Fatal("expected tampered ciphertext to fail")
	}
}
