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

// RestTokenRevokeService is the core operation PUT /v2/session/revoke delegates to.
type RestTokenRevokeService interface {
	Exec(ctx context.Context, request *core.TokenRevokeRequest) error
}

// tokenRevokeErrors maps a refused sign-out to its status. A refresh token of another user answers
// like one this service never issued.
var tokenRevokeErrors = httpf.ErrMap{
	core.ErrTokenRevokeInvalidRefreshToken: http.StatusForbidden,
	core.ErrTokenRevokeMismatchClaims:      http.StatusForbidden,
	core.ErrInvalidRequest:                 http.StatusUnprocessableEntity,
}

// RestTokenRevokeRequest is the JSON body of PUT /v2/session/revoke.
type RestTokenRevokeRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// RestTokenRevoke serves PUT /v2/session/revoke, which signs a session out by revoking its refresh
// token. The caller's access token names the user the refresh token must belong to. It answers an
// empty 204, also when the token was already revoked or expired.
type RestTokenRevoke struct {
	service RestTokenRevokeService
	logger  logging.Log
}

func NewRestTokenRevoke(service RestTokenRevokeService, logger logging.Log) *RestTokenRevoke {
	return &RestTokenRevoke{service: service, logger: logger}
}

func (handler *RestTokenRevoke) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.TokenRevoke")
	defer span.End()

	claims, err := session.MustGetClaimsContext(ctx)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusUnauthorized}, err)

		return
	}

	var request RestTokenRevokeRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.TokenRevokeRequest{
		UserID:       lo.FromPtr(claims.UserID),
		RefreshToken: request.RefreshToken,
	}

	err = handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, tokenRevokeErrors, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
