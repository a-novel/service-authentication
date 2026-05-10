package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.credentialsUpdatePassword.sql
var credentialsUpdatePassword string

// ErrCredentialsUpdatePasswordNotFound is returned by
// [CredentialsUpdatePassword.Exec] when no row matches the requested ID. It is
// joined onto the underlying sql.ErrNoRows so callers can branch on it with
// errors.Is.
var ErrCredentialsUpdatePasswordNotFound = errors.New("credentials not found")

// CredentialsUpdatePasswordRequest is the input to [CredentialsUpdatePassword.Exec].
type CredentialsUpdatePasswordRequest struct {
	// ID of the credentials to update.
	ID uuid.UUID
	// Password is the Argon2id hash to persist, not the plaintext. Plaintext
	// must be hashed by the caller (typically via lib.GenerateArgon2) before
	// reaching this layer; storing the hash means a database leak does not
	// reveal usable credentials.
	Password string
	// Now is the timestamp recorded as the row's update time.
	Now time.Time
}

// CredentialsUpdatePassword updates the password used by a set of credentials.
// Make sure the user identity has been properly validated BEFORE calling this
// repository.
type CredentialsUpdatePassword struct{}

func NewCredentialsUpdatePassword() *CredentialsUpdatePassword {
	return &CredentialsUpdatePassword{}
}

func (repository *CredentialsUpdatePassword) Exec(
	ctx context.Context, request *CredentialsUpdatePasswordRequest,
) (*Credentials, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.CredentialsUpdatePassword")
	defer span.End()

	span.SetAttributes(
		attribute.String("credentials.id", request.ID.String()),
		attribute.String("credentials.password", strings.Repeat("*", len(request.Password))),
		attribute.Int64("credentials.now", request.Now.Unix()),
	)

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	entity := new(Credentials)

	err = tx.NewRaw(credentialsUpdatePassword, request.Password, request.Now, request.ID).Scan(ctx, entity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = errors.Join(err, ErrCredentialsUpdatePasswordNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
