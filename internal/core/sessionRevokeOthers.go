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

// SessionRevokeOthersDao signs out every session of the caller but one.
type SessionRevokeOthersDao interface {
	Exec(ctx context.Context, request *dao.SessionRevokeOthersRequest) error
}

// SessionRevokeOthersRequest names the caller, and the session they keep.
type SessionRevokeOthersRequest struct {
	// UserID is the caller, taken from their verified access token.
	UserID uuid.UUID
	// SessionID is the caller's own session, which stays live.
	SessionID string
}

// SessionRevokeOthers signs out every live session of the caller but their own, at once and on every
// platform, as [SessionRevoke] signs out one.
type SessionRevokeOthers struct {
	dao SessionRevokeOthersDao
	// refreshTokenLeeway is the clock-skew tolerance the verifier grants an expired refresh token.
	refreshTokenLeeway time.Duration
}

func NewSessionRevokeOthers(dao SessionRevokeOthersDao, refreshTokenLeeway time.Duration) *SessionRevokeOthers {
	return &SessionRevokeOthers{dao: dao, refreshTokenLeeway: refreshTokenLeeway}
}

func (service *SessionRevokeOthers) Exec(ctx context.Context, request *SessionRevokeOthersRequest) error {
	ctx, span := otel.Tracer().Start(ctx, "core.SessionRevokeOthers")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.id", request.UserID.String()),
		attribute.String("session.id", request.SessionID),
	)

	keepID, err := uuid.Parse(request.SessionID)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("parse session ID: %w", err))
	}

	revocation := &dao.SessionRevokeOthersRequest{
		UserID: request.UserID,
		KeepID: keepID,
		Leeway: service.refreshTokenLeeway,
		Now:    time.Now(),
	}

	err = service.dao.Exec(ctx, revocation)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("revoke sessions: %w", err))
	}

	return nil
}
