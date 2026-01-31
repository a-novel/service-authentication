package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/services"
)

type TokenRefreshService interface {
	Exec(ctx context.Context, request *services.TokenRefreshRequest) (*services.Token, error)
}

type TokenRefreshRequest struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type TokenRefresh struct {
	service TokenRefreshService
	logger  logging.Log
}

func NewTokenRefresh(service TokenRefreshService, logger logging.Log) *TokenRefresh {
	return &TokenRefresh{service: service, logger: logger}
}

func (handler *TokenRefresh) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "handler.TokenRefresh")
	defer span.End()

	decoder := json.NewDecoder(r.Body)

	var request TokenRefreshRequest

	err := decoder.Decode(&request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	res, err := handler.service.Exec(ctx, &services.TokenRefreshRequest{
		AccessToken:  request.AccessToken,
		RefreshToken: request.RefreshToken,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
			services.ErrTokenRefreshInvalidAccessToken:  http.StatusForbidden,
			services.ErrTokenRefreshInvalidRefreshToken: http.StatusForbidden,
			services.ErrTokenRefreshMismatchClaims:      http.StatusForbidden,
			services.ErrTokenRefreshMismatchSource:      http.StatusForbidden,
			services.ErrInvalidRequest:                  http.StatusUnprocessableEntity,
		}, err)

		return
	}

	httpf.SendJSON(ctx, w, span, loadToken(res))
}
