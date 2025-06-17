package lib

import (
	"context"
	"errors"
	"fmt"
	"os"
)

var ErrNewAgoraContext = errors.New("NewAgoraContext")

func NewErrNewAgoraContext(err error) error {
	return errors.Join(err, ErrNewAgoraContext)
}

func NewAgoraContext(parentCTX context.Context) (context.Context, error) {
	ctx, err := NewMasterKeyContext(parentCTX)
	if err != nil {
		return nil, NewErrNewAgoraContext(fmt.Errorf("create master key context: %w", err))
	}

	ctx, err = NewPostgresContext(ctx, os.Getenv("DSN"))
	if err != nil {
		return nil, NewErrNewAgoraContext(fmt.Errorf("create postgres context: %w", err))
	}

	return ctx, nil
}
