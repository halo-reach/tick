package credential

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
)

func Encrypt(plaintext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func Decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func MaskValue(val string) string {
	if len(val) < 8 {
		return "****"
	}
	return val[:3] + "****" + val[len(val)-3:]
}

func GeneratePreview(configJSON []byte) json.RawMessage {
	var m map[string]interface{}
	if err := json.Unmarshal(configJSON, &m); err != nil {
		return json.RawMessage(`{}`)
	}

	for k, v := range m {
		if s, ok := v.(string); ok {
			m[k] = MaskValue(s)
		}
	}

	out, _ := json.Marshal(m)
	return out
}
