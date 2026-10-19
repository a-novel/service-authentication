package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/lib"
)

// TokenCreateService is the core operation PUT /v2/session delegates to.
type TokenCreateService interface {
	Exec(ctx context.Context, request *core.TokenCreateRequest) (*core.Token, error)
}

// TokenCreateRequest is the JSON body of PUT /v2/session.
type TokenCreateRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// TokenCreate serves PUT /v2/session, which signs a user in.
type TokenCreate struct {
	service TokenCreateService
	logger  logging.Log
}

func NewTokenCreate(service TokenCreateService, logger logging.Log) *TokenCreate {
	return &TokenCreate{service: service, logger: logger}
}

func (handler *TokenCreate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.TokenCreate")
	defer span.End()

	var request TokenCreateRequest

	err := httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	res, err := handler.service.Exec(ctx, &core.TokenCreateRequest{
		Email:    request.Email,
		Password: request.Password,
	})
	if err != nil {
		// Both "email not found" and "invalid password" return 401 to prevent email enumeration.
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
			core.ErrCredentialsByEmailNotFound: http.StatusUnauthorized,
			lib.ErrInvalidPassword:             http.StatusUnauthorized,
			core.ErrInvalidRequest:             http.StatusUnprocessableEntity,
		}, err)

		return
	}

	httpf.SendJSONStatus(ctx, w, span, http.StatusOK, loadToken(res))
}
