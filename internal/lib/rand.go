package lib

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// URLCharList is the alphabet of URL-safe characters that NewRandomURLString draws
// from when generating random strings.
var URLCharList = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// Alphabet length hoisted out of the generation loop.
var (
	urlCharListLen = int64(len(URLCharList))
)

// NewRandomURLString generates a random, URL-safe string of the given length.
func NewRandomURLString(length int) (string, error) {
	out := make([]rune, length)

	for i := range out {
		num, err := rand.Int(rand.Reader, big.NewInt(urlCharListLen))
		if err != nil {
			return "", fmt.Errorf("generate random URL: %w", err)
		}

		out[i] = URLCharList[num.Int64()]
	}

	return string(out), nil
}
