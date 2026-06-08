package credential

import (
	"crypto/rand"
	"encoding/hex"
)

func generateID(prefix string, randBytes int) string {
	b := make([]byte, randBytes)
	rand.Read(b)
	return prefix + hex.EncodeToString(b)
}
