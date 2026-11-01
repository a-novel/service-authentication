package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/uptrace/bun"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.refreshTokenRevocationExist.sql
var refreshTokenRevocationExistQuery string

// RefreshTokenRevocationExistRequest is the input to [PgRefreshTokenRevocationExist.Exec].
type RefreshTokenRevocationExistRequest struct {
	// JTIs are the [RefreshTokenRevocation.JTI] values to look up.
	JTIs []string
}

// PgRefreshTokenRevocationExist reports whether any of the given JTIs was revoked. JTIs never revoked
// yield false with a nil error.
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

	res, err := db.NewRaw(refreshTokenRevocationExistQuery, bun.List(request.JTIs)).Exec(ctx)
	if err != nil {
		return false, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	n, err := res.RowsAffected()
	if err != nil {
		return false, otel.ReportError(span, fmt.Errorf("get rows affected: %w", err))
	}

	return n > 0, nil
}
