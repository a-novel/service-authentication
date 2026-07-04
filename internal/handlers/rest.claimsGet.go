package handlers

import (
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/handlers/middlewares"
)

// ClaimsGet is the REST handler that returns the caller's own claims, read from
// the request context where the authentication middleware deposits them. It
// responds 403 when the request carries no resolved claims.
type ClaimsGet struct {
	logger logging.Log
}

// NewClaimsGet returns a ClaimsGet handler.
func NewClaimsGet(logger logging.Log) *ClaimsGet {
	return &ClaimsGet{logger: logger}
}

func (handler *ClaimsGet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ClaimsGet")
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
