package lib

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"

	"github.com/a-novel-kit/context"
)

type masterKeyContext struct{}

const (
	MasterKeyEnv = "MASTER_KEY"
)

var ErrNoMasterKey = errors.New("missing master key")

func NewMasterKeyContext(ctx context.Context) (context.Context, error) {
	masterKeyRaw := os.Getenv(MasterKeyEnv)
	if masterKeyRaw == "" {
		return ctx, ErrNoMasterKey
	}

	masterKeyBytes, err := hex.DecodeString(masterKeyRaw)
	if err != nil {
		return ctx, fmt.Errorf("(NewMasterKeyContext) decode master key: %w", err)
	}

	var masterKey [32]byte

	copy(masterKey[:], masterKeyBytes)

	return context.WithValue(ctx, masterKeyContext{}, masterKey), nil
}

func MasterKeyContext(ctx context.Context) ([32]byte, error) {
	return context.ExtractValue[[32]byte](ctx, masterKeyContext{})
}
