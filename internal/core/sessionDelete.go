package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/dao"
)

// ErrSessionActive is returned by [SessionDelete.Exec] for a session still active. The caller signs
// it out first.
var ErrSessionActive = errors.New("session is active")

// SessionDeleteDaoSelect loads the session to remove, among the caller's.
type SessionDeleteDaoSelect interface {
	Exec(ctx context.Context, request *dao.SessionSelectRequest) (*dao.Session, error)
}

// SessionDeleteDao removes the row of a logged-out session.
type SessionDeleteDao interface {
	Exec(ctx context.Context, request *dao.SessionDeleteRequest) error
}

// SessionDeleteRequest names the session to remove, and the caller it must belong to.
type SessionDeleteRequest struct {
	// UserID is the caller, taken from their verified access token.
	UserID uuid.UUID
	// ID is the session to remove.
	ID uuid.UUID
}

// SessionDelete removes a logged-out session from the caller's devices. An active session returns
// [ErrSessionActive], since removing it would hide a live session from its account. An unknown
// session, or another account's, returns [ErrSessionNotFound]. A removed session stays signed out:
// its revocation stays until its own purge.
type SessionDelete struct {
	dao       SessionDeleteDao
	daoSelect SessionDeleteDaoSelect
}

func NewSessionDelete(dao SessionDeleteDao, daoSelect SessionDeleteDaoSelect) *SessionDelete {
	return &SessionDelete{dao: dao, daoSelect: daoSelect}
}

func (service *SessionDelete) Exec(ctx context.Context, request *SessionDeleteRequest) error {
	ctx, span := otel.Tracer().Start(ctx, "core.SessionDelete")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.id", request.UserID.String()),
		attribute.String("session.id", request.ID.String()),
	)

	selectRequest := &dao.SessionSelectRequest{ID: request.ID, UserID: request.UserID}

	entity, err := service.daoSelect.Exec(ctx, selectRequest)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("select session: %w", err))
	}

	if !entity.LoggedOut {
		return otel.ReportError(span, ErrSessionActive)
	}

	// A logged-out session never becomes active again, so the delete misses only when a concurrent
	// request removed the row first.
	err = service.dao.Exec(ctx, &dao.SessionDeleteRequest{ID: request.ID, UserID: request.UserID})
	if errors.Is(err, dao.ErrSessionDeleteNotFound) {
		err = errors.Join(err, ErrSessionNotFound)
	}

	if err != nil {
		return otel.ReportError(span, fmt.Errorf("delete session: %w", err))
	}

	return nil
}
