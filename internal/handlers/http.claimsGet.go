package handlers

import (
	"errors"
	"net/http"

	"github.com/a-novel/golib/httpf"
	"github.com/a-novel/golib/otel"

	"github.com/a-novel/service-authentication/internal/handlers/middlewares"
)

type ClaimsGet struct{}

func NewClaimsGet() *ClaimsGet {
	return new(ClaimsGet)
}

func (handler *ClaimsGet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "handler.ClaimsGet")
	defer span.End()

	claims, err := middlewares.GetClaimsContext(ctx)
	if err != nil {
		httpf.HandleError(ctx, w, span, nil, err)

		return
	}

	if claims == nil {
		httpf.HandleError(ctx, w, span, httpf.ErrMap{nil: http.StatusForbidden}, errors.New("claims not found"))

		return
	}

	httpf.SendJSON(ctx, w, span, Claims{
		UserID:         claims.UserID,
		Roles:          claims.Roles,
		RefreshTokenID: claims.RefreshTokenID,
	})
}
