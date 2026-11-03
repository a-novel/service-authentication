package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.credentialsUpdateLocationConsent.sql
var credentialsUpdateLocationConsentQuery string

// ErrCredentialsUpdateLocationConsentNotFound is returned by [PgCredentialsUpdateLocationConsent.Exec]
// when no row matches the requested ID.
var ErrCredentialsUpdateLocationConsentNotFound = errors.New("credentials not found")

// CredentialsUpdateLocationConsentRequest is the input to [PgCredentialsUpdateLocationConsent.Exec].
type CredentialsUpdateLocationConsentRequest struct {
	// ID of the credentials to update.
	ID uuid.UUID
	// See [Credentials.LocationConsent].
	LocationConsent bool
}

// PgCredentialsUpdateLocationConsent sets whether an account records the location of its sessions.
type PgCredentialsUpdateLocationConsent struct{}

func NewPgCredentialsUpdateLocationConsent() *PgCredentialsUpdateLocationConsent {
	return &PgCredentialsUpdateLocationConsent{}
}

func (dao *PgCredentialsUpdateLocationConsent) Exec(
	ctx context.Context, request *CredentialsUpdateLocationConsentRequest,
) (*Credentials, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgCredentialsUpdateLocationConsent")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &Credentials{}

	err = db.NewRaw(credentialsUpdateLocationConsentQuery, request.LocationConsent, request.ID).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrCredentialsUpdateLocationConsentNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
