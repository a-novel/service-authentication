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

//go:embed pg.credentialsUpdateEmail.sql
var credentialsUpdateEmailQuery string

var (
	// ErrCredentialsUpdateEmailAlreadyExists is returned by [PgCredentialsUpdateEmail.Exec] when another
	// account registered the new email between the short code's issuance and its consumption.
	ErrCredentialsUpdateEmailAlreadyExists = errors.New("credentials already exists")
	// ErrCredentialsUpdateEmailNotFound is returned by [PgCredentialsUpdateEmail.Exec] when no row
	// matches the requested ID.
	ErrCredentialsUpdateEmailNotFound = errors.New("credentials not found")
)

// CredentialsUpdateEmailRequest is the input to [PgCredentialsUpdateEmail.Exec].
type CredentialsUpdateEmailRequest struct {
	// ID of the credentials to update.
	ID uuid.UUID
	// Email is the new address to assign. The caller verifies its ownership first, with a short
	// code emailed to it.
	Email string
	// Now is the timestamp recorded as the row's update time.
	Now time.Time
}

// PgCredentialsUpdateEmail assigns a new email address to a credentials row.
type PgCredentialsUpdateEmail struct{}

func NewPgCredentialsUpdateEmail() *PgCredentialsUpdateEmail {
	return &PgCredentialsUpdateEmail{}
}

func (dao *PgCredentialsUpdateEmail) Exec(
	ctx context.Context, request *CredentialsUpdateEmailRequest,
) (*Credentials, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgCredentialsUpdateEmail")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &Credentials{}

	err = db.NewRaw(credentialsUpdateEmailQuery, request.Email, request.Now, request.ID).Scan(ctx, entity)
	if isUniqueViolation(err) {
		err = errors.Join(err, ErrCredentialsUpdateEmailAlreadyExists)
	}

	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrCredentialsUpdateEmailNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
