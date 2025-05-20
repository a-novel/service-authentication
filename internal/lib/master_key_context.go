package lib

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
)

type masterKeyContext struct{}

const (
	MasterKeyEnv = "MASTER_KEY"
)

var (
	ErrInvalidMasterKey = errors.New("invalid master key")
	ErrNoMasterKey      = errors.New("missing master key")
)

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
	masterKey, ok := ctx.Value(masterKeyContext{}).([32]byte)

	if !ok {
		return [32]byte{}, fmt.Errorf(
			"(MasterKeyContext) extract master key: %w: got type %T, expected %T",
			ErrInvalidMasterKey,
			ctx.Value(masterKeyContext{}), [32]byte{},
		)
	}

	return masterKey, nil
}
