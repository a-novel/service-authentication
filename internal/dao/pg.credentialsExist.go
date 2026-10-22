package dao

import (
	"context"
	_ "embed"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.credentialsExist.sql
var credentialsExistQuery string

// CredentialsExistRequest is the input to [PgCredentialsExist.Exec].
type CredentialsExistRequest struct {
	// Email to check for an existing registration. Comparison is case-sensitive.
	Email string
}

// PgCredentialsExist reports whether a user with the given email address exists.
//
// A missing user yields false with a nil error.
type PgCredentialsExist struct{}

func NewPgCredentialsExist() *PgCredentialsExist {
	return &PgCredentialsExist{}
}

func (dao *PgCredentialsExist) Exec(ctx context.Context, request *CredentialsExistRequest) (bool, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgCredentialsExist")
	defer span.End()

	span.SetAttributes(attribute.String("credentials.email", request.Email))

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return false, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	res, err := db.NewRaw(credentialsExistQuery, request.Email).Exec(ctx)
	if err != nil {
		return false, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	n, err := res.RowsAffected()
	if err != nil {
		return false, otel.ReportError(span, fmt.Errorf("get rows affected: %w", err))
	}

	return n >= 1, nil
}
