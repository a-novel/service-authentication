package handlers

import (
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/handlers/middlewares"
)

type ClaimsGet struct {
	logger logging.Log
}

func NewClaimsGet(logger logging.Log) *ClaimsGet {
	return &ClaimsGet{logger: logger}
}

func (handler *ClaimsGet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "handler.ClaimsGet")
	defer span.End()

	claims, err := middlewares.MustGetClaimsContext(ctx)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusForbidden}, err)

		return
	}

	httpf.SendJSON(ctx, w, span, Claims{
		UserID:         claims.UserID,
		Roles:          claims.Roles,
		RefreshTokenID: claims.RefreshTokenID,
	})
}
