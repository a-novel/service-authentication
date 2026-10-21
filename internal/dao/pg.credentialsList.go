package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/samber/lo"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.credentialsList.sql
var credentialsListQuery string

// CredentialsListRequest is the input to [PgCredentialsList.Exec].
type CredentialsListRequest struct {
	Limit  int
	Offset int
	// Roles, if non-empty, restricts the result to credentials whose role is in
	// the slice. An empty slice returns credentials of every role.
	Roles []string
	// Emails, if non-empty, restricts the result to exact, case-sensitive addresses.
	Emails []string
}

// PgCredentialsList returns a page of credentials, newest first. It leaves [Credentials.Password]
// empty.
type PgCredentialsList struct{}

func NewPgCredentialsList() *PgCredentialsList {
	return &PgCredentialsList{}
}

func (dao *PgCredentialsList) Exec(
	ctx context.Context, request *CredentialsListRequest,
) ([]*Credentials, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgCredentialsList")
	defer span.End()

	span.SetAttributes(
		attribute.Int("data.limit", request.Limit),
		attribute.Int("data.offset", request.Offset),
		attribute.StringSlice("data.roles", request.Roles),
	)

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entities := make([]*Credentials, 0, request.Limit)

	err = db.NewRaw(
		credentialsListQuery,
		bun.NullZero(request.Limit),
		request.Offset,
		// bun.List fails on a nil slice. An empty one renders as NULL, which the query reads as no filter.
		bun.List(lo.CoalesceSliceOrEmpty(request.Roles)),
		bun.List(lo.CoalesceSliceOrEmpty(request.Emails)),
	).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entities, nil
}
