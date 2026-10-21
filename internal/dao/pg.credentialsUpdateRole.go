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

// ErrCredentialsUpdateRoleNotFound is returned by [PgCredentialsUpdateRole.Exec] when no row matches
// the requested ID.
var ErrCredentialsUpdateRoleNotFound = errors.New("credentials not found")

// CredentialsUpdateRoleRequest is the input to [PgCredentialsUpdateRole.Exec].
type CredentialsUpdateRoleRequest struct {
	// ID of the credentials to update.
	ID uuid.UUID
	// Role is the new role name to assign. The core layer validates it against the permission
	// configuration.
	Role string
	// Now is the timestamp recorded as the row's update time.
	Now time.Time
}

// PgCredentialsUpdateRole assigns a new role to a credentials row.
type PgCredentialsUpdateRole struct{}

func NewPgCredentialsUpdateRole() *PgCredentialsUpdateRole {
	return &PgCredentialsUpdateRole{}
}

func (dao *PgCredentialsUpdateRole) Exec(
	ctx context.Context, request *CredentialsUpdateRoleRequest,
) (*Credentials, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgCredentialsUpdateRole")
	defer span.End()

	span.SetAttributes(attribute.String("credentials.id", request.ID.String()))

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &Credentials{}

	err = db.NewRaw(credentialsUpdateRoleQuery, request.Role, request.Now, request.ID).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrCredentialsUpdateRoleNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
