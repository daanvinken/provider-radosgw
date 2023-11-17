package utils

import (
	"math/rand"
	"os"
	"time"
)

const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func Getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func GenerateRandomSecret(length int) string {
	rand.Seed(time.Now().UnixNano())
	secret := make([]byte, length)

	for i := 0; i < length; i++ {
		secret[i] = charset[rand.Intn(len(charset))]
	}
	return string(secret)
}
