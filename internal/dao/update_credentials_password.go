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

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"
)

//go:embed update_credentials_password.sql
var updateCredentialsPasswordQuery string

// UpdateCredentialsPasswordData is the input used to perform the
// UpdateCredentialsPasswordRepository.UpdateCredentialsPassword action.
type UpdateCredentialsPasswordData struct {
	// The user's password, encrypted using scrypt. This will be used when creating a session.
	//
	// This value is sensitive, and MUST always be transmitted using a secure channel / encryption. It MUST NEVER be
	// stored clear in the database.
	Password string
	// Time at which the credentials were updated.
	Now time.Time
}

// UpdateCredentialsPasswordRepository is the repository used to perform the
// UpdateCredentialsPasswordRepository.UpdateCredentialsPassword action.
//
// You may create one using the NewUpdateCredentialsPasswordRepository function.
type UpdateCredentialsPasswordRepository struct{}

func NewUpdateCredentialsPasswordRepository() *UpdateCredentialsPasswordRepository {
	return &UpdateCredentialsPasswordRepository{}
}

// UpdateCredentialsPassword updates the password of a set of credentials in the database.
//
// This method MUST be called after extra checks on the user identity have been performed. This is a sensitive
// operation.
func (repository *UpdateCredentialsPasswordRepository) UpdateCredentialsPassword(
	ctx context.Context, userID uuid.UUID, data UpdateCredentialsPasswordData,
) (*CredentialsEntity, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.UpdateCredentialsPassword")
	defer span.End()

	span.SetAttributes(
		attribute.String("credentials.id", userID.String()),
		attribute.Int64("credentials.now", data.Now.Unix()),
	)

	// Retrieve a connection to postgres from the context.
	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres client: %w", err))
	}

	entity := &CredentialsEntity{}

	err = tx.NewRaw(updateCredentialsPasswordQuery, data.Password, data.Now, userID).Scan(ctx, entity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, otel.ReportError(span, ErrCredentialsNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("update credentials: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
