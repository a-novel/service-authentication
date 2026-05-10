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

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.credentialsUpdateRole.sql
var credentialsUpdateRoleQuery string

// ErrCredentialsUpdateRoleNotFound is returned by [CredentialsUpdateRole.Exec]
// when no row matches the requested ID. It is joined onto the underlying
// sql.ErrNoRows so callers can branch on it with errors.Is.
var ErrCredentialsUpdateRoleNotFound = errors.New("credentials not found")

// CredentialsUpdateRoleRequest is the input to [CredentialsUpdateRole.Exec].
type CredentialsUpdateRoleRequest struct {
	// ID of the credentials to update.
	ID uuid.UUID
	// Role is the new role name to assign. The repository does not validate the
	// value against the configured permission map; that check belongs to the
	// service layer.
	Role string
	// Now is the timestamp recorded as the row's update time.
	Now time.Time
}

// CredentialsUpdateRole assigns a new role to a set of credentials.
type CredentialsUpdateRole struct{}

func NewCredentialsUpdateRole() *CredentialsUpdateRole {
	return &CredentialsUpdateRole{}
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

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	entity := new(Credentials)

	err = tx.NewRaw(credentialsUpdateRoleQuery, request.Role, request.Now, request.ID).Scan(ctx, entity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = errors.Join(err, ErrCredentialsUpdateRoleNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
