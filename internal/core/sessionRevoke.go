package core

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/dao"
)

// SessionRevokeDao loads the session to sign out, among the caller's.
type SessionRevokeDao interface {
	Exec(ctx context.Context, request *dao.SessionSelectRequest) (*dao.Session, error)
}

// SessionRevokeDaoRevocationInsert records the sign-out.
type SessionRevokeDaoRevocationInsert interface {
	Exec(ctx context.Context, request *dao.RefreshTokenRevocationInsertRequest) error
}

// SessionRevokeRequest names the session to sign out, and the caller it must belong to.
type SessionRevokeRequest struct {
	// UserID is the caller, taken from their verified access token.
	UserID uuid.UUID
	// ID is the session to sign out.
	ID uuid.UUID
}

// SessionRevoke signs one of the caller's sessions out, the caller's own included, on every
// platform: no refresh token of the session renews it, and [ClaimsGet] refuses its access tokens.
// Signing out a session already signed out succeeds. A session that ended, or that belongs to another
// account, returns [ErrSessionNotFound], as an unknown one does.
type SessionRevoke struct {
	dao                 SessionRevokeDao
	daoRevocationInsert SessionRevokeDaoRevocationInsert
	// refreshTokenLeeway is the clock-skew tolerance the verifier grants an expired refresh token.
	refreshTokenLeeway time.Duration
}

func NewSessionRevoke(
	dao SessionRevokeDao, daoRevocationInsert SessionRevokeDaoRevocationInsert, refreshTokenLeeway time.Duration,
) *SessionRevoke {
	return &SessionRevoke{dao: dao, daoRevocationInsert: daoRevocationInsert, refreshTokenLeeway: refreshTokenLeeway}
}

func (service *SessionRevoke) Exec(ctx context.Context, request *SessionRevokeRequest) error {
	ctx, span := otel.Tracer().Start(ctx, "core.SessionRevoke")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.id", request.UserID.String()),
		attribute.String("session.id", request.ID.String()),
	)

	entity, err := service.dao.Exec(ctx, &dao.SessionSelectRequest{ID: request.ID, UserID: request.UserID})
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("select session: %w", err))
	}

	now := time.Now()

	if !entity.ExpiresAt.After(now) {
		return otel.ReportError(span, fmt.Errorf("session ended: %w", ErrSessionNotFound))
	}

	// The verifier accepts the session's refresh tokens until it ends, plus the leeway, so the
	// revocation lasts as long.
	revocation := &dao.RefreshTokenRevocationInsertRequest{
		JTIs:      []string{entity.ID.String()},
		ExpiresAt: entity.ExpiresAt.Add(service.refreshTokenLeeway),
		Now:       now,
	}

	err = service.daoRevocationInsert.Exec(ctx, revocation)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("revoke session: %w", err))
	}

	return nil
}
