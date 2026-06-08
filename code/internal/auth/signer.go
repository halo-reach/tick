package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func GenerateSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate secret: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func Sign(taskID, timestamp, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(taskID + timestamp + secret))
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func Verify(taskID, timestamp, secret, signature string) bool {
	expected := Sign(taskID, timestamp, secret)
	return hmac.Equal([]byte(expected), []byte(signature))
}
