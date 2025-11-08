package lib

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
)

var ErrNewRandomURLString = errors.New("NewRandomURLString")

func NewErrNewRandomURLString(err error) error {
	return errors.Join(err, ErrNewRandomURLString)
}

// URLCharList is a list of URL-valid characters, for generating random strings.
var URLCharList = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func NewRandomURLString(length int) (string, error) {
	out := make([]rune, length)

	for i := range out {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(URLCharList))))
		if err != nil {
			return "", NewErrNewRandomURLString(fmt.Errorf("generate random index: %w", err))
		}

		out[i] = URLCharList[num.Int64()]
	}

	return string(out), nil
}
