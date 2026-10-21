package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.credentialsUpdatePassword.sql
var credentialsUpdatePasswordQuery string

// ErrCredentialsUpdatePasswordNotFound is returned by [PgCredentialsUpdatePassword.Exec] when no row
// matches the requested ID.
var ErrCredentialsUpdatePasswordNotFound = errors.New("credentials not found")

// CredentialsUpdatePasswordRequest is the input to [PgCredentialsUpdatePassword.Exec].
type CredentialsUpdatePasswordRequest struct {
	// ID of the credentials to update.
	ID uuid.UUID
	// Password is the Argon2id hash to persist. The caller hashes the plaintext, so a database leak
	// yields no usable credentials.
	Password string
	// Now is the timestamp recorded as the row's update time.
	Now time.Time
}

// PgCredentialsUpdatePassword replaces the password hash of a credentials row. The caller verifies the
// user's identity first.
type PgCredentialsUpdatePassword struct{}

func NewPgCredentialsUpdatePassword() *PgCredentialsUpdatePassword {
	return &PgCredentialsUpdatePassword{}
}

func (dao *PgCredentialsUpdatePassword) Exec(
	ctx context.Context, request *CredentialsUpdatePasswordRequest,
) (*Credentials, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgCredentialsUpdatePassword")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &Credentials{}

	err = db.NewRaw(credentialsUpdatePasswordQuery, request.Password, request.Now, request.ID).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrCredentialsUpdatePasswordNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
