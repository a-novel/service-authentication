package dao

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun/driver/pgdriver"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"
)

//go:embed pg.credentialsInsert.sql
var credentialsInsertQuery string

var ErrCredentialsInsertAlreadyExists = errors.New("credentials already exists")

type CredentialsInsertRequest struct {
	// See Credentials.ID.
	ID uuid.UUID
	// See Credentials.Email.
	Email string
	// See Credentials.Password.
	Password string
	// See Credentials.Role.
	Role string
	// Time used for user registration.
	Now time.Time
}

// CredentialsInsert inserts a new set of credentials into the database.
//
// The new credentials must be unique, otherwise ErrCredentialsInsertAlreadyExists is thrown.
type CredentialsInsert struct{}

func NewCredentialsInsert() *CredentialsInsert {
	return new(CredentialsInsert)
}

func (repository *CredentialsInsert) Exec(
	ctx context.Context, request *CredentialsInsertRequest,
) (*Credentials, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.CredentialsInsert")
	defer span.End()

	span.SetAttributes(
		attribute.String("credentials.id", request.ID.String()),
		attribute.String("credentials.email", request.Email),
		attribute.String("credentials.password", strings.Repeat("*", len(request.Password))),
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
