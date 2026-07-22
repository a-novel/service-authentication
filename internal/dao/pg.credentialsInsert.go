package dao

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun/driver/pgdriver"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.credentialsInsert.sql
var credentialsInsertQuery string

// ErrCredentialsInsertAlreadyExists is returned by [CredentialsInsert.Exec] when
// the email is already registered. It is detected from the Postgres unique-violation
// SQLSTATE (23505) and joined onto the underlying driver error so callers can
// branch on it with errors.Is.
var ErrCredentialsInsertAlreadyExists = errors.New("credentials already exists")

// CredentialsInsertRequest is the input to [CredentialsInsert.Exec].
type CredentialsInsertRequest struct {
	// See Credentials.ID.
	ID uuid.UUID
	// See Credentials.Email.
	Email string
	// See Credentials.Password.
	Password string
	// See Credentials.Role.
	Role string
	// Now is the timestamp recorded as the row's creation time.
	Now time.Time
}

// CredentialsInsert inserts a new set of credentials into the database. The email
// must be unique; a duplicate returns [ErrCredentialsInsertAlreadyExists].
type CredentialsInsert struct{}

func NewCredentialsInsert() *CredentialsInsert {
	return &CredentialsInsert{}
}

func (dao *CredentialsInsert) Exec(
	ctx context.Context, request *CredentialsInsertRequest,
) (*Credentials, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.CredentialsInsert")
	defer span.End()

	span.SetAttributes(
		attribute.String("credentials.id", request.ID.String()),
		attribute.String("credentials.email", request.Email),
		// The password never goes on the span. A redaction still carries its length.
		attribute.String("credentials.role", request.Role),
		attribute.Int64("credentials.now", request.Now.Unix()),
	)

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	entity := new(Credentials)

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
		var pgErr pgdriver.Error
		if errors.As(err, &pgErr) && pgErr.Field('C') == "23505" {
			err = errors.Join(err, ErrCredentialsInsertAlreadyExists)
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
