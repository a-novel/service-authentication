package lib

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// URLCharList is a list of URL-valid characters, for generating random strings.
var URLCharList = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// Precompute values to optimize the algorithm.
var (
	urlCharListLen = int64(len(URLCharList))
)

// NewRandomURLString generates a random, url-safe text of a given length.
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
