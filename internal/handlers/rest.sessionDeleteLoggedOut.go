package handlers

import (
	"context"
	"net/http"

	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

// RestSessionDeleteLoggedOutService is the core operation DELETE /v2/sessions/logged-out delegates to.
type RestSessionDeleteLoggedOutService interface {
	Exec(ctx context.Context, request *core.SessionDeleteLoggedOutRequest) error
}

// RestSessionDeleteLoggedOut serves DELETE /v2/sessions/logged-out, which removes every logged-out
// session from the caller's devices. It answers an empty 204, with or without a session to remove.
type RestSessionDeleteLoggedOut struct {
	service RestSessionDeleteLoggedOutService
	logger  logging.Log
}

func NewRestSessionDeleteLoggedOut(
	service RestSessionDeleteLoggedOutService, logger logging.Log,
) *RestSessionDeleteLoggedOut {
	return &RestSessionDeleteLoggedOut{service: service, logger: logger}
}

func (handler *RestSessionDeleteLoggedOut) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.SessionDeleteLoggedOut")
	defer span.End()

	claims, err := session.RequireClaimsContext(ctx)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, nil, err)

		return
	}

	err = handler.service.Exec(ctx, &core.SessionDeleteLoggedOutRequest{UserID: lo.FromPtr(claims.UserID)})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, nil, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
