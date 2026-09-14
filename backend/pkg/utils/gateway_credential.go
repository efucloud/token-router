package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	gatewayCredentialVersion   = "v1"
	gatewayCredentialAAD       = "token-router:gateway-credential:v1"
	gatewayCredentialMasterKey = "tjXNPZBQzu5qG1Z0B+zt9fhHZ7bz44OwSd4ZsCXGS0k="
)

func decodeGatewayMasterKey() ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(gatewayCredentialMasterKey)
	if err != nil {
		return nil, fmt.Errorf("decode built-in gateway credential key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("built-in gateway credential key must decode to 32 bytes, got %d", len(key))
	}
	return key, nil
}

// EncryptGatewayCredential encrypts an upstream credential with AES-256-GCM.
// The result contains a format version and a random nonce and is safe to store.
func EncryptGatewayCredential(plaintext string) (string, error) {
	key, err := decodeGatewayMasterKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create gateway credential cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gateway credential gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate gateway credential nonce: %w", err)
	}
	sealed := gcm.Seal(nil, nonce, []byte(plaintext), []byte(gatewayCredentialAAD))
	payload := append(nonce, sealed...)
	return gatewayCredentialVersion + ":" + base64.RawURLEncoding.EncodeToString(payload), nil
}

// DecryptGatewayCredential decrypts a value created by EncryptGatewayCredential.
func DecryptGatewayCredential(encrypted string) (string, error) {
	version, payloadText, found := strings.Cut(encrypted, ":")
	if !found || version != gatewayCredentialVersion {
		return "", errors.New("unsupported gateway credential format")
	}
	key, err := decodeGatewayMasterKey()
	if err != nil {
		return "", err
	}
	payload, err := base64.RawURLEncoding.DecodeString(payloadText)
	if err != nil {
		return "", errors.New("invalid gateway credential payload")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create gateway credential cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gateway credential gcm: %w", err)
	}
	if len(payload) < gcm.NonceSize()+gcm.Overhead() {
		return "", errors.New("invalid gateway credential payload")
	}
	nonce, ciphertext := payload[:gcm.NonceSize()], payload[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, []byte(gatewayCredentialAAD))
	if err != nil {
		return "", errors.New("gateway credential authentication failed")
	}
	return string(plaintext), nil
}
