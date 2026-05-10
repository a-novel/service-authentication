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

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.credentialsUpdateEmail.sql
var credentialsUpdateEmailQuery string

var (
	// ErrCredentialsUpdateEmailAlreadyExists is returned by
	// [CredentialsUpdateEmail.Exec] when the new email is already registered to
	// another user. It is detected from the Postgres unique-violation SQLSTATE
	// (23505) and joined onto the driver error so callers can branch on it.
	// This race can occur when a different account claims the email between
	// short-code issuance and consumption.
	ErrCredentialsUpdateEmailAlreadyExists = errors.New("credentials already exists")
	// ErrCredentialsUpdateEmailNotFound is returned by [CredentialsUpdateEmail.Exec]
	// when no row matches the requested ID. It is joined onto the underlying
	// sql.ErrNoRows.
	ErrCredentialsUpdateEmailNotFound = errors.New("credentials not found")
)

// CredentialsUpdateEmailRequest is the input to [CredentialsUpdateEmail.Exec].
type CredentialsUpdateEmailRequest struct {
	// ID of the credentials to update.
	ID uuid.UUID
	// Email is the new address to assign. Caller must verify ownership (typically
	// via a short code emailed to this address) before invoking the repository.
	Email string
	// Now is the timestamp recorded as the row's update time.
	Now time.Time
}

// CredentialsUpdateEmail updates the email address of a set of credentials. The new address should be validated
// beforehand.
type CredentialsUpdateEmail struct{}

func NewCredentialsUpdateEmail() *CredentialsUpdateEmail {
	return &CredentialsUpdateEmail{}
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
