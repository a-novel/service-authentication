package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/core"
)

// RestTokenRevokeService is the core operation PUT /v2/session/revoke delegates to.
type RestTokenRevokeService interface {
	Exec(ctx context.Context, request *core.TokenRevokeRequest) error
}

// tokenRevokeErrors maps a refused sign-out to its status.
var tokenRevokeErrors = httpf.ErrMap{
	core.ErrTokenRevokeInvalidRefreshToken: http.StatusForbidden,
	core.ErrInvalidRequest:                 http.StatusUnprocessableEntity,
}

// RestTokenRevokeRequest is the JSON body of PUT /v2/session/revoke.
type RestTokenRevokeRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// RestTokenRevoke serves PUT /v2/session/revoke, which signs a session out by revoking its refresh
// token. It answers an empty 204, also when the token was already revoked or expired.
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

	var request RestTokenRevokeRequest

	err := httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	err = handler.service.Exec(ctx, &core.TokenRevokeRequest{RefreshToken: request.RefreshToken})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, tokenRevokeErrors, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
