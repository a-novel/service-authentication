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

//go:embed update_credentials_email.sql
var updateCredentialsEmailQuery string

// UpdateCredentialsEmailData is the input used to perform the
// UpdateCredentialsEmailRepository.UpdateCredentialsEmail action.
type UpdateCredentialsEmailData struct {
	// Email of the credentials.
	//
	// Each email MUST be unique among all credentials. You may use the ExistsCredentialsEmailRepository to ensure
	// a given address is not already in use.
	//
	// Once an email is updated, the old value is freed and can be used by another set of credentials.
	Email string
	// Time at which the credentials were updated.
	Now time.Time
}

// UpdateCredentialsEmailRepository is the repository used to perform the
// UpdateCredentialsEmailRepository.UpdateCredentialsEmail action.
//
// You may create one using the NewUpdateCredentialsEmailRepository function.
type UpdateCredentialsEmailRepository struct{}

func NewUpdateCredentialsEmailRepository() *UpdateCredentialsEmailRepository {
	return &UpdateCredentialsEmailRepository{}
}

// UpdateCredentialsEmail updates the email of a set of credentials in the database.
//
// The new email MUST have been validated beforehand, and be unique among all credentials.
// The ExistsCredentialsEmailRepository can be used beforehand to ensure an email is available.
//
// Once the update completes, the old email is freed and can be used by another set of credentials.
func (repository *UpdateCredentialsEmailRepository) UpdateCredentialsEmail(
	ctx context.Context, userID uuid.UUID, data UpdateCredentialsEmailData,
) (*CredentialsEntity, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.UpdateCredentialsEmail")
	defer span.End()

	span.SetAttributes(
		attribute.String("credentials.id", userID.String()),
		attribute.String("credentials.email", data.Email),
		attribute.Int64("credentials.now", data.Now.Unix()),
	)

	// Retrieve a connection to postgres from the context.
	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres client: %w", err))
	}

	entity := &CredentialsEntity{}

	// Execute query.
	err = tx.NewRaw(updateCredentialsEmailQuery, data.Email, data.Now, userID).Scan(ctx, entity)
	if err != nil {
		var pgErr pgdriver.Error
		if errors.As(err, &pgErr) && pgErr.Field('C') == "23505" {
			return nil, otel.ReportError(span, errors.Join(err, ErrCredentialsAlreadyExists))
		}

		if errors.Is(err, sql.ErrNoRows) {
			return nil, otel.ReportError(span, ErrCredentialsNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("update credentials: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
