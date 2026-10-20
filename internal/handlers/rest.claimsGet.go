package handlers

import (
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/handlers/middlewares"
)

// RestClaimsGet is the REST handler that returns the caller's own claims, read from
// the request context where the authentication middleware deposits them. It
// responds 403 when the request carries no resolved claims.
type RestClaimsGet struct {
	logger logging.Log
}

func NewRestClaimsGet(logger logging.Log) *RestClaimsGet {
	return &RestClaimsGet{logger: logger}
}

func (handler *RestClaimsGet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ClaimsGet")
	defer span.End()

	claims, err := middlewares.MustGetClaimsContext(ctx)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusForbidden}, err)

		return
	}

	response := Claims{
		UserID:         claims.UserID,
		Roles:          claims.Roles,
		RefreshTokenID: claims.RefreshTokenID,
	}

	httpf.SendJSONStatus(ctx, w, span, http.StatusOK, response)
}
