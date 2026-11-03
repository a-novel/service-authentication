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

//go:embed pg.credentialsLocationConsent.sql
var credentialsLocationConsentQuery string

// ErrCredentialsLocationConsentNotFound is returned by [PgCredentialsLocationConsent.Exec] when no row
// matches the requested ID.
var ErrCredentialsLocationConsentNotFound = errors.New("credentials not found")

// CredentialsLocationConsentRequest is the input to [PgCredentialsLocationConsent.Exec].
type CredentialsLocationConsentRequest struct {
	// ID of the credentials to read.
	ID uuid.UUID
}

// PgCredentialsLocationConsent reads an account's [Credentials.LocationConsent], and locks it against
// a consent update until the caller's transaction ends. Run it in the transaction that writes what the
// consent decides.
type PgCredentialsLocationConsent struct{}

func NewPgCredentialsLocationConsent() *PgCredentialsLocationConsent {
	return &PgCredentialsLocationConsent{}
}

func (dao *PgCredentialsLocationConsent) Exec(
	ctx context.Context, request *CredentialsLocationConsentRequest,
) (bool, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgCredentialsLocationConsent")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return false, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var consent bool

	err = db.NewRaw(credentialsLocationConsentQuery, request.ID).Scan(ctx, &consent)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrCredentialsLocationConsentNotFound)
	}

	if err != nil {
		return false, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return consent, nil
}
