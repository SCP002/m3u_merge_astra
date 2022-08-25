package rnd

import (
	"math/rand"
	"time"
)

// String returns random string
func String(length uint, uppercase bool, numbers bool) string {
	rand.Seed(time.Now().UnixNano())

	charset := "abcdefghijklmnopqrstuvwxyz"
	if uppercase {
		charset += "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	}
	if numbers {
		charset += "0123456789"
	}
	charsetRunes := []rune(charset)

	outRunes := make([]rune, length)

	for i := range outRunes {
		outRunes[i] = charsetRunes[rand.Intn(len(charsetRunes))]
	}

	return string(outRunes)
}
