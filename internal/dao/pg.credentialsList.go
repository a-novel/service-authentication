package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.credentialsList.sql
var credentialsListQuery string

// CredentialsListRequest is the input to [CredentialsList.Exec]. Pagination uses
// limit/offset; an empty Roles slice disables the role filter and returns
// credentials of every role.
type CredentialsListRequest struct {
	Limit  int
	Offset int
	// Roles, if non-empty, restricts the result to credentials whose role is in
	// the slice. An empty slice returns credentials of every role.
	Roles []string
}

// CredentialsList returns a set of paginated credentials from the database. Only the public fields are returned,
// private authentication information is left empty.
type CredentialsList struct{}

func NewCredentialsList() *CredentialsList {
	return &CredentialsList{}
}

func (dao *CredentialsList) Exec(
	ctx context.Context, request *CredentialsListRequest,
) ([]*Credentials, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.CredentialsList")
	defer span.End()

	span.SetAttributes(
		attribute.Int("data.limit", request.Limit),
		attribute.Int("data.offset", request.Offset),
		attribute.StringSlice("data.roles", request.Roles),
	)

	if len(request.Roles) == 0 {
		// Make sure roles are defined in the query, to prevent type error.
		// An empty array will still ignore roles filter like a nil value is expected to do.
		request.Roles = []string{}
	}

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	entities := make([]*Credentials, 0, request.Limit)

	err = tx.NewRaw(
		credentialsListQuery,
		bun.NullZero(request.Limit),
		request.Offset,
		bun.List(request.Roles),
	).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, entities), nil
}
