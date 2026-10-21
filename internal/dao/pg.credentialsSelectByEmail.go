package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.credentialsSelectByEmail.sql
var credentialsSelectByEmailQuery string

// ErrCredentialsSelectByEmailNotFound is returned by [PgCredentialsSelectByEmail.Exec] when no row
// matches the requested email.
var ErrCredentialsSelectByEmailNotFound = errors.New("credentials not found")

// CredentialsSelectByEmailRequest is the input to [PgCredentialsSelectByEmail.Exec].
type CredentialsSelectByEmailRequest struct {
	// Email of the credentials to fetch. Comparison is case-sensitive.
	Email string
}

// PgCredentialsSelectByEmail fetches a single credentials row by email. [PgCredentialsSelect] looks one
// up by ID.
type PgCredentialsSelectByEmail struct{}

func NewPgCredentialsSelectByEmail() *PgCredentialsSelectByEmail {
	return &PgCredentialsSelectByEmail{}
}

func (dao *PgCredentialsSelectByEmail) Exec(
	ctx context.Context, request *CredentialsSelectByEmailRequest,
) (*Credentials, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgCredentialsSelectByEmail")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &Credentials{}

	err = db.NewRaw(credentialsSelectByEmailQuery, request.Email).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrCredentialsSelectByEmailNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
