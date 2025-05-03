package lib

import (
	"errors"
	"fmt"

	"github.com/a-novel-kit/context"
	pgctx "github.com/a-novel-kit/context/pgbun"

	"github.com/a-novel/service-authentication/migrations"
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

	ctx, err = pgctx.NewContext(ctx, &migrations.Migrations)
	if err != nil {
		return nil, NewErrNewAgoraContext(fmt.Errorf("create postgres context: %w", err))
	}

	return ctx, nil
}
