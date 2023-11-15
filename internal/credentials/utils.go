package credentials

import (
	"k8s.io/apimachinery/pkg/util/rand"
	"time"
)

const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func GenerateRandomSecret(length int) string {
	rand.Seed(time.Now().UnixNano())
	secret := make([]byte, length)

	for i := 0; i < length; i++ {
		secret[i] = charset[rand.Intn(len(charset))]
	}
	return string(secret)
}
