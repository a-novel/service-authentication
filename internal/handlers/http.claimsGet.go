package handlers

import (
	"net/http"

	"github.com/a-novel/golib/httpf"
	"github.com/a-novel/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/handlers/middlewares"
)

type ClaimsGet struct{}

func NewClaimsGet() *ClaimsGet {
	return new(ClaimsGet)
}

func (handler *ClaimsGet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "handler.ClaimsGet")
	defer span.End()

	claims, err := middlewares.MustGetClaimsContext(ctx)
	if err != nil {
		httpf.HandleError(ctx, w, span, httpf.ErrMap{nil: http.StatusForbidden}, err)

		return
	}

	httpf.SendJSON(ctx, w, span, Claims{
		UserID:         claims.UserID,
		Roles:          claims.Roles,
		RefreshTokenID: claims.RefreshTokenID,
	})
}
