package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"
)

//go:embed pg.credentialsSelectByEmail.sql
var credentialsSelectByEmailQuery string

var ErrCredentialsSelectByEmailNotFound = errors.New("credentials not found")

type CredentialsSelectByEmailRequest struct {
	Email string
}

type CredentialsSelectByEmail struct{}

func NewCredentialsSelectByEmail() *CredentialsSelectByEmail {
	return new(CredentialsSelectByEmail)
}

func (repository *CredentialsSelectByEmail) Exec(
	ctx context.Context, request *CredentialsSelectByEmailRequest,
) (*Credentials, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.CredentialsSelectByEmail")
	defer span.End()

	span.SetAttributes(attribute.String("email", request.Email))

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	entity := new(Credentials)

	err = tx.NewRaw(credentialsSelectByEmailQuery, request.Email).Scan(ctx, entity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = errors.Join(err, ErrCredentialsSelectByEmailNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
