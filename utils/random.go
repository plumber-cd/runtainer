package utils

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/plumber-cd/runtainer/log"
)

func RandomHex(n int) string {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		log.Normal.Panic(err)
	}
	return hex.EncodeToString(bytes)
}
