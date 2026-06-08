package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

const keyPrefix = "tk_live_"

func GenerateAPIKey() (plaintext, hash, prefix string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", "", fmt.Errorf("generate key: %w", err)
	}
	plaintext = keyPrefix + hex.EncodeToString(b)
	hash = HashKey(plaintext)
	prefix = plaintext[:8]
	return
}

func HashKey(plaintext string) string {
	h := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(h[:])
}
