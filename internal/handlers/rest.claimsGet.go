package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

// RestClaimsGetService is the core operation GET /v2/session delegates to.
type RestClaimsGetService interface {
	Exec(ctx context.Context, request *core.ClaimsGetRequest) (*session.Claims, error)
}

// claimsGetErrors answers 401 to a revoked session, as the auth middleware does to a rejected token.
var claimsGetErrors = httpf.ErrMap{
	core.ErrClaimsGetRevoked: http.StatusUnauthorized,
}

// RestClaimsGet is the REST handler that returns the caller's own claims, read from
// the request context where the authentication middleware deposits them. It
// responds 403 when the request carries no resolved claims, and 401 once their
// session is revoked.
type RestClaimsGet struct {
	service RestClaimsGetService
	logger  logging.Log
}

func NewRestClaimsGet(service RestClaimsGetService, logger logging.Log) *RestClaimsGet {
	return &RestClaimsGet{service: service, logger: logger}
}

func (handler *RestClaimsGet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ClaimsGet")
	defer span.End()

	claims, err := session.MustGetClaimsContext(ctx)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusForbidden}, err)

		return
	}

	claims, err = handler.service.Exec(ctx, &core.ClaimsGetRequest{Claims: claims})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, claimsGetErrors, err)

		return
	}

	response := Claims{
		UserID:         claims.UserID,
		Roles:          claims.Roles,
		RefreshTokenID: claims.RefreshTokenID,
		SessionID:      claims.Session(),
		SessionExp:     claims.SessionExp,
		ClientID:       claims.ClientID,
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, response)
}
