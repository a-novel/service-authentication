package dao

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.authorizationCodeInsert.sql
var authorizationCodeInsertQuery string

// AuthorizationCodeInsertRequest is the input to [PgAuthorizationCodeInsert.Exec].
type AuthorizationCodeInsertRequest struct {
	// See [AuthorizationCode.ID].
	ID []byte
	// See [AuthorizationCode.UserID].
	UserID uuid.UUID

	// See [AuthorizationCode.ClientID].
	ClientID string
	// See [AuthorizationCode.RedirectURI].
	RedirectURI string
	// See [AuthorizationCode.CodeChallenge].
	CodeChallenge string

	// See [AuthorizationCode.SessionID].
	SessionID string
	// See [AuthorizationCode.SessionExpiresAt].
	SessionExpiresAt time.Time

	// See [AuthorizationCode.ExpiresAt].
	ExpiresAt time.Time
}

// PgAuthorizationCodeInsert stores a new, unconsumed authorization code.
type PgAuthorizationCodeInsert struct{}

func NewPgAuthorizationCodeInsert() *PgAuthorizationCodeInsert {
	return &PgAuthorizationCodeInsert{}
}

func (dao *PgAuthorizationCodeInsert) Exec(ctx context.Context, request *AuthorizationCodeInsertRequest) error {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgAuthorizationCodeInsert")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	_, err = db.NewRaw(
		authorizationCodeInsertQuery,
		request.ID,
		request.UserID,
		request.ClientID,
		request.RedirectURI,
		request.CodeChallenge,
		request.SessionID,
		request.SessionExpiresAt,
		request.ExpiresAt,
	).Exec(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return nil
}
