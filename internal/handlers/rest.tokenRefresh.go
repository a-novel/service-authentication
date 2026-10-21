package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/core"
)

// RestTokenRefreshService is the core operation PATCH /v2/session delegates to.
type RestTokenRefreshService interface {
	Exec(ctx context.Context, request *core.TokenRefreshRequest) (*core.Token, error)
}

// tokenRefreshErrors maps a refused session renewal to its status.
var tokenRefreshErrors = httpf.ErrMap{
	core.ErrTokenRefreshInvalidAccessToken:  http.StatusForbidden,
	core.ErrTokenRefreshInvalidRefreshToken: http.StatusForbidden,
	core.ErrTokenRefreshMismatchClaims:      http.StatusForbidden,
	core.ErrTokenRefreshMismatchSource:      http.StatusForbidden,
	// The account behind a still-valid refresh token is gone, so the caller signs in again.
	core.ErrCredentialsGetNotFound: http.StatusUnauthorized,
	core.ErrInvalidRequest:         http.StatusUnprocessableEntity,
}

// RestTokenRefreshRequest is the JSON body of PATCH /v2/session.
type RestTokenRefreshRequest struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

// RestTokenRefresh serves PATCH /v2/session, which renews a token pair.
type RestTokenRefresh struct {
	service RestTokenRefreshService
	logger  logging.Log
}

func NewRestTokenRefresh(service RestTokenRefreshService, logger logging.Log) *RestTokenRefresh {
	return &RestTokenRefresh{service: service, logger: logger}
}

func (handler *RestTokenRefresh) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.TokenRefresh")
	defer span.End()

	var request RestTokenRefreshRequest

	err := httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.TokenRefreshRequest{
		AccessToken:  request.AccessToken,
		RefreshToken: request.RefreshToken,
	}

	res, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, tokenRefreshErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadToken(res))
}
