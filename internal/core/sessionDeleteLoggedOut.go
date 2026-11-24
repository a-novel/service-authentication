package core

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/dao"
)

// SessionDeleteLoggedOutDao removes the row of every logged-out session of the caller.
type SessionDeleteLoggedOutDao interface {
	Exec(ctx context.Context, request *dao.SessionDeleteLoggedOutRequest) error
}

// SessionDeleteLoggedOutRequest names the caller whose logged-out sessions to remove.
type SessionDeleteLoggedOutRequest struct {
	// UserID is the caller, taken from their verified access token.
	UserID uuid.UUID
}

// SessionDeleteLoggedOut removes every logged-out session from the caller's devices at once, as
// [SessionDelete] removes one. Active sessions stay listed, and a caller without a logged-out session
// gets no error. The removed sessions stay signed out: each revocation stays until its own purge.
type SessionDeleteLoggedOut struct {
	dao SessionDeleteLoggedOutDao
}

func NewSessionDeleteLoggedOut(dao SessionDeleteLoggedOutDao) *SessionDeleteLoggedOut {
	return &SessionDeleteLoggedOut{dao: dao}
}

func (service *SessionDeleteLoggedOut) Exec(ctx context.Context, request *SessionDeleteLoggedOutRequest) error {
	ctx, span := otel.Tracer().Start(ctx, "core.SessionDeleteLoggedOut")
	defer span.End()

	span.SetAttributes(attribute.String("user.id", request.UserID.String()))

	err := service.dao.Exec(ctx, &dao.SessionDeleteLoggedOutRequest{UserID: request.UserID})
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("delete logged-out sessions: %w", err))
	}

	return nil
}
