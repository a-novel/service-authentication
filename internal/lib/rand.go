package lib

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// urlCharList is the alphabet of URL-safe characters that NewRandomURLString draws from. It is
// ASCII, so a byte indexes one character.
const urlCharList = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// urlCharListLen bounds each random draw. rand.Int only reads it, so every call shares it.
var urlCharListLen = big.NewInt(int64(len(urlCharList)))

// NewRandomURLString generates a random, URL-safe string of the given length.
func NewRandomURLString(length int) (string, error) {
	out := make([]byte, length)

	for i := range out {
		num, err := rand.Int(rand.Reader, urlCharListLen)
		if err != nil {
			return "", fmt.Errorf("generate random URL: %w", err)
		}

		out[i] = urlCharList[num.Int64()]
	}

	return string(out), nil
}
