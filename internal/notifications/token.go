package notifications

import (
	"crypto/rand"
	"encoding/hex"
)

type GenerateTokenFunc func() (string, error)

func generateToken() (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	token := hex.EncodeToString(tokenBytes)
	return token, nil
}
