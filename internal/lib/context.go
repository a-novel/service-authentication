package lib

import (
	"context"
	"errors"
	"fmt"
)

var ErrNewAgoraContext = errors.New("NewAgoraContext")

func NewErrNewAgoraContext(err error) error {
	return errors.Join(err, ErrNewAgoraContext)
}

func NewAgoraContext(parentCTX context.Context, dsn string) (context.Context, error) {
	ctx, err := NewMasterKeyContext(parentCTX)
	if err != nil {
		return parentCTX, NewErrNewAgoraContext(fmt.Errorf("create master key context: %w", err))
	}

	ctx, err = NewPostgresContext(ctx, dsn)
	if err != nil {
		return parentCTX, NewErrNewAgoraContext(fmt.Errorf("create postgres context: %w", err))
	}

	return ctx, nil
}
