package dao

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.credentialsInsert.sql
var credentialsInsertQuery string

// ErrCredentialsInsertAlreadyExists is returned by [PgCredentialsInsert.Exec] when the email is
// already registered.
var ErrCredentialsInsertAlreadyExists = errors.New("credentials already exists")

// CredentialsInsertRequest is the input to [PgCredentialsInsert.Exec].
type CredentialsInsertRequest struct {
	// See [Credentials.ID].
	ID uuid.UUID
	// See [Credentials.Email].
	Email string
	// See [Credentials.Password].
	Password string
	// See [Credentials.Role].
	Role string
	// Now is the instant stored as both created_at and updated_at.
	Now time.Time
}

// PgCredentialsInsert inserts a new set of credentials into the database. The email
// must be unique; a duplicate returns [ErrCredentialsInsertAlreadyExists].
type PgCredentialsInsert struct{}

func NewPgCredentialsInsert() *PgCredentialsInsert {
	return &PgCredentialsInsert{}
}

func (dao *PgCredentialsInsert) Exec(
	ctx context.Context, request *CredentialsInsertRequest,
) (*Credentials, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgCredentialsInsert")
	defer span.End()

	span.SetAttributes(
		attribute.String("credentials.id", request.ID.String()),
		// The password never goes on the span. A redaction still carries its length.
		attribute.String("credentials.role", request.Role),
	)

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	entity := &Credentials{}

	err = tx.NewRaw(
		credentialsInsertQuery,
		request.ID,
		request.Email,
		request.Password,
		request.Now,
		request.Now,
		request.Role,
	).Scan(ctx, entity)
	if err != nil {
		if isUniqueViolation(err) {
			err = errors.Join(err, ErrCredentialsInsertAlreadyExists)
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
