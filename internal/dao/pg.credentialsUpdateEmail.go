package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun/driver/pgdriver"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"
)

//go:embed pg.credentialsUpdateEmail.sql
var credentialsUpdateEmailQuery string

var (
	ErrCredentialsUpdateEmailAlreadyExists = errors.New("credentials already exists")
	ErrCredentialsUpdateEmailNotFound      = errors.New("credentials not found")
)

type CredentialsUpdateEmailRequest struct {
	// The ID of the credentials to update.
	ID uuid.UUID
	// The new Email value for the selected credentials.
	Email string
	// Time used as modification date.
	Now time.Time
}

// CredentialsUpdateEmail updates the email address of a set of credentials. The new address should be validated
// beforehand.
type CredentialsUpdateEmail struct{}

func NewCredentialsUpdateEmail() *CredentialsUpdateEmail {
	return new(CredentialsUpdateEmail)
}

func (repository *CredentialsUpdateEmail) Exec(
	ctx context.Context, request *CredentialsUpdateEmailRequest,
) (*Credentials, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.CredentialsUpdateEmail")
	defer span.End()

	span.SetAttributes(
		attribute.String("credentials.id", request.ID.String()),
		attribute.String("credentials.email", request.Email),
		attribute.Int64("credentials.now", request.Now.Unix()),
	)

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	entity := new(Credentials)

	err = tx.NewRaw(credentialsUpdateEmailQuery, request.Email, request.Now, request.ID).Scan(ctx, entity)
	if err != nil {
		var pgErr pgdriver.Error
		if errors.As(err, &pgErr) && pgErr.Field('C') == "23505" {
			// Can happen if the email was taken by another account during the verification process.
			err = errors.Join(err, ErrCredentialsUpdateEmailAlreadyExists)
		} else if errors.Is(err, sql.ErrNoRows) {
			err = errors.Join(err, ErrCredentialsUpdateEmailNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
