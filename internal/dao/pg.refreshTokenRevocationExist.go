package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.refreshTokenRevocationExist.sql
var refreshTokenRevocationExistQuery string

// RefreshTokenRevocationExistRequest is the input to [PgRefreshTokenRevocationExist.Exec].
type RefreshTokenRevocationExistRequest struct {
	// See [RefreshTokenRevocation.JTI].
	JTI string
}

// PgRefreshTokenRevocationExist reports whether a refresh token was revoked. A token never revoked
// yields false with a nil error.
type PgRefreshTokenRevocationExist struct{}

func NewPgRefreshTokenRevocationExist() *PgRefreshTokenRevocationExist {
	return &PgRefreshTokenRevocationExist{}
}

func (dao *PgRefreshTokenRevocationExist) Exec(
	ctx context.Context, request *RefreshTokenRevocationExistRequest,
) (bool, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgRefreshTokenRevocationExist")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return false, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	res, err := db.NewRaw(refreshTokenRevocationExistQuery, request.JTI).Exec(ctx)
	if err != nil {
		return false, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	n, err := res.RowsAffected()
	if err != nil {
		return false, otel.ReportError(span, fmt.Errorf("get rows affected: %w", err))
	}

	return n > 0, nil
}
