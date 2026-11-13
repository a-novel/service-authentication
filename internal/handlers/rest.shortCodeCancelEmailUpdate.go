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

// RestShortCodeCancelEmailUpdateService is the core operation DELETE /v2/short-code/update-email
// delegates to.
type RestShortCodeCancelEmailUpdateService interface {
	Exec(ctx context.Context, request *core.ShortCodeCancelEmailUpdateRequest) error
}

// shortCodeCancelEmailUpdateErrors maps a refused cancellation to its status.
var shortCodeCancelEmailUpdateErrors = httpf.ErrMap{
	core.ErrShortCodeNotFound: http.StatusNotFound,
}

// RestShortCodeCancelEmailUpdate serves DELETE /v2/short-code/update-email, which cancels the caller's
// email change awaiting confirmation. It answers an empty 204.
type RestShortCodeCancelEmailUpdate struct {
	service RestShortCodeCancelEmailUpdateService
	logger  logging.Log
}

func NewRestShortCodeCancelEmailUpdate(
	service RestShortCodeCancelEmailUpdateService, logger logging.Log,
) *RestShortCodeCancelEmailUpdate {
	return &RestShortCodeCancelEmailUpdate{service: service, logger: logger}
}

func (handler *RestShortCodeCancelEmailUpdate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ShortCodeCancelEmailUpdate")
	defer span.End()

	claims, err := session.MustGetClaimsContext(ctx)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, nil, err)

		return
	}

	err = handler.service.Exec(ctx, &core.ShortCodeCancelEmailUpdateRequest{UserID: lo.FromPtr(claims.UserID)})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, shortCodeCancelEmailUpdateErrors, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
