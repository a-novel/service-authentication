package handlers

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

// RestSessionRevokeService is the core operation PUT /v2/sessions/{sessionID}/revoke delegates to.
type RestSessionRevokeService interface {
	Exec(ctx context.Context, request *core.SessionRevokeRequest) error
}

// sessionRevokeErrors maps a refused sign-out to its status. Another account's session answers like
// an unknown one.
var sessionRevokeErrors = httpf.ErrMap{core.ErrSessionNotFound: http.StatusNotFound}

// RestSessionRevoke serves PUT /v2/sessions/{sessionID}/revoke, which signs one of the caller's sessions
// out, their own included. It answers an empty 204, also when the session was already signed out.
type RestSessionRevoke struct {
	service RestSessionRevokeService
	logger  logging.Log
}

func NewRestSessionRevoke(service RestSessionRevokeService, logger logging.Log) *RestSessionRevoke {
	return &RestSessionRevoke{service: service, logger: logger}
}

func (handler *RestSessionRevoke) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.SessionRevoke")
	defer span.End()

	claims, err := session.MustGetClaimsContext(ctx)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, nil, err)

		return
	}

	sessionID, err := uuid.Parse(chi.URLParam(r, "sessionID"))
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.SessionRevokeRequest{
		UserID: lo.FromPtr(claims.UserID),
		ID:     sessionID,
	}

	err = handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, sessionRevokeErrors, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
