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

// RestSessionDeleteService is the core operation DELETE /v2/sessions/{sessionID} delegates to.
type RestSessionDeleteService interface {
	Exec(ctx context.Context, request *core.SessionDeleteRequest) error
}

// sessionDeleteErrors maps a refused removal to its status. Another account's session answers like
// an unknown one.
var sessionDeleteErrors = httpf.ErrMap{
	core.ErrSessionNotFound: http.StatusNotFound,
	core.ErrSessionActive:   http.StatusConflict,
}

// RestSessionDelete serves DELETE /v2/sessions/{sessionID}, which removes a logged-out session from
// the caller's devices. It answers an empty 204.
type RestSessionDelete struct {
	service RestSessionDeleteService
	logger  logging.Log
}

func NewRestSessionDelete(service RestSessionDeleteService, logger logging.Log) *RestSessionDelete {
	return &RestSessionDelete{service: service, logger: logger}
}

func (handler *RestSessionDelete) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.SessionDelete")
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

	serviceRequest := &core.SessionDeleteRequest{
		UserID: lo.FromPtr(claims.UserID),
		ID:     sessionID,
	}

	err = handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, sessionDeleteErrors, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
