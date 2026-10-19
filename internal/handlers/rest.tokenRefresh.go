package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/core"
)

// TokenRefreshService is the core operation PATCH /v2/session delegates to.
type TokenRefreshService interface {
	Exec(ctx context.Context, request *core.TokenRefreshRequest) (*core.Token, error)
}

// TokenRefreshRequest is the JSON body of PATCH /v2/session.
type TokenRefreshRequest struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

// TokenRefresh serves PATCH /v2/session, which renews a token pair.
type TokenRefresh struct {
	service TokenRefreshService
	logger  logging.Log
}

func NewTokenRefresh(service TokenRefreshService, logger logging.Log) *TokenRefresh {
	return &TokenRefresh{service: service, logger: logger}
}

func (handler *TokenRefresh) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.TokenRefresh")
	defer span.End()

	var request TokenRefreshRequest

	err := httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	res, err := handler.service.Exec(ctx, &core.TokenRefreshRequest{
		AccessToken:  request.AccessToken,
		RefreshToken: request.RefreshToken,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
			core.ErrTokenRefreshInvalidAccessToken:  http.StatusForbidden,
			core.ErrTokenRefreshInvalidRefreshToken: http.StatusForbidden,
			core.ErrTokenRefreshMismatchClaims:      http.StatusForbidden,
			core.ErrTokenRefreshMismatchSource:      http.StatusForbidden,
			// The credentials behind a still-valid refresh token were deleted — re-authenticate.
			core.ErrCredentialsGetNotFound: http.StatusUnauthorized,
			core.ErrInvalidRequest:         http.StatusUnprocessableEntity,
		}, err)

		return
	}

	httpf.SendJSONStatus(ctx, w, span, http.StatusOK, loadToken(res))
}
