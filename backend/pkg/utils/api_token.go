package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"strings"
)

const (
	APITokenPrefix       = "tr_"
	apiTokenRandomBytes  = 32
	apiTokenDisplayChars = 12
)

// GenerateAPIToken returns the one-time plaintext, its SHA-256 hex digest and
// a non-secret prefix suitable for management lists and logs.
func GenerateAPIToken() (plaintext, hash, displayPrefix string, err error) {
	random := make([]byte, apiTokenRandomBytes)
	if _, err = io.ReadFull(rand.Reader, random); err != nil {
		return "", "", "", err
	}
	plaintext = APITokenPrefix + base64.RawURLEncoding.EncodeToString(random)
	hash, err = HashAPIToken(plaintext)
	if err != nil {
		return "", "", "", err
	}
	displayPrefix = plaintext
	if len(displayPrefix) > apiTokenDisplayChars {
		displayPrefix = displayPrefix[:apiTokenDisplayChars]
	}
	return plaintext, hash, displayPrefix, nil
}

func HashAPIToken(token string) (string, error) {
	if !strings.HasPrefix(token, APITokenPrefix) || len(token) <= len(APITokenPrefix) {
		return "", errors.New("invalid API key format")
	}
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:]), nil
}

func APITokenMatches(token, expectedHash string) bool {
	actualHash, err := HashAPIToken(token)
	if err != nil || len(actualHash) != len(expectedHash) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(actualHash), []byte(expectedHash)) == 1
}
