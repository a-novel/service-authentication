package lib

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// URLCharList is a list of URL-valid characters, for generating random strings.
var URLCharList = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func NewRandomURLString(length int) (string, error) {
	out := make([]rune, length)

	for i := range out {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(URLCharList))))
		if err != nil {
			return "", fmt.Errorf("(NewRandomURLString) generate random index: %w", err)
		}

		out[i] = URLCharList[num.Int64()]
	}

	return string(out), nil
}
