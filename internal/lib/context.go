package lib

import (
	"context"
	"errors"
	"fmt"

	"github.com/a-novel/service-authentication/migrations"
)

var ErrNewAgoraContext = errors.New("NewAgoraContext")

func NewErrNewAgoraContext(err error) error {
	return errors.Join(err, ErrNewAgoraContext)
}

func NewAgoraContext(parentCTX context.Context, dsn string) (context.Context, error) {
	ctx, err := NewPostgresContext(parentCTX, dsn, migrations.Migrations)
	if err != nil {
		return parentCTX, NewErrNewAgoraContext(fmt.Errorf("create postgres context: %w", err))
	}

	return ctx, nil
}
