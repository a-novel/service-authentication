package core

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/dao"
)

// SessionListLimit bounds the sessions [SessionList] returns.
const SessionListLimit = 100

// SessionListDao loads the caller's sessions.
type SessionListDao interface {
	Exec(ctx context.Context, request *dao.SessionListRequest) ([]*dao.Session, error)
}

// SessionListRequest names the caller whose sessions to list.
type SessionListRequest struct {
	// UserID is the caller, taken from their verified access token.
	UserID uuid.UUID
	// SessionID is the caller's own session, which the list marks current.
	SessionID string
}

// SessionList returns the caller's sessions, [SessionListLimit] at most: the active ones first, then
// the logged-out ones, each group latest activity first.
type SessionList struct {
	dao SessionListDao
}

func NewSessionList(dao SessionListDao) *SessionList {
	return &SessionList{dao: dao}
}

func (service *SessionList) Exec(ctx context.Context, request *SessionListRequest) ([]*Session, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.SessionList")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.id", request.UserID.String()),
		attribute.String("session.id", request.SessionID),
	)

	entities, err := service.dao.Exec(ctx, &dao.SessionListRequest{UserID: request.UserID, Limit: SessionListLimit})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list sessions: %w", err))
	}

	return lo.Map(entities, func(entity *dao.Session, _ int) *Session {
		return newSession(entity, request.SessionID)
	}), nil
}
