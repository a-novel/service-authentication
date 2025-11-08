package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"
)

//go:embed pg.credentialsUpdatePassword.sql
var credentialsUpdatePassword string

var ErrCredentialsUpdatePasswordNotFound = errors.New("credentials not found")

type CredentialsUpdatePasswordRequest struct {
	ID       uuid.UUID
	Password string
	Now      time.Time
}

type CredentialsUpdatePassword struct{}

func NewCredentialsUpdatePassword() *CredentialsUpdatePassword {
	return new(CredentialsUpdatePassword)
}

func (repository *CredentialsUpdatePassword) Exec(
	ctx context.Context, request *CredentialsUpdatePasswordRequest,
) (*Credentials, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.CredentialsUpdatePassword")
	defer span.End()

	span.SetAttributes(
		attribute.String("credentials.id", request.ID.String()),
		attribute.String("credentials.password", strings.Repeat("*", len(request.Password))),
		attribute.Int64("credentials.now", request.Now.Unix()),
	)

	// Retrieve a connection to postgres from the context.
	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	entity := new(Credentials)

	err = tx.NewRaw(credentialsUpdatePassword, request.Password, request.Now, request.ID).Scan(ctx, entity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, otel.ReportError(span, errors.Join(err, ErrCredentialsUpdatePasswordNotFound))
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
