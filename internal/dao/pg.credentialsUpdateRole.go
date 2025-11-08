package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"
)

//go:embed pg.credentialsUpdateRole.sql
var credentialsUpdateRoleQuery string

var ErrCredentialsUpdateRoleNotFound = errors.New("credentials not found")

type CredentialsUpdateRoleRequest struct {
	ID   uuid.UUID
	Role string
	Now  time.Time
}

type CredentialsUpdateRole struct{}

func NewCredentialsUpdateRole() *CredentialsUpdateRole {
	return new(CredentialsUpdateRole)
}

func (repository *CredentialsUpdateRole) Exec(
	ctx context.Context, request *CredentialsUpdateRoleRequest,
) (*Credentials, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.CredentialsUpdateRole")
	defer span.End()

	span.SetAttributes(
		attribute.String("credentials.id", request.ID.String()),
		attribute.String("credentials.role", request.Role),
		attribute.Int64("credentials.now", request.Now.Unix()),
	)

	// Retrieve a connection to postgres from the context.
	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	entity := new(Credentials)

	err = tx.NewRaw(credentialsUpdateRoleQuery, request.Role, request.Now, request.ID).Scan(ctx, entity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, otel.ReportError(span, errors.Join(err, ErrCredentialsUpdateRoleNotFound))
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
