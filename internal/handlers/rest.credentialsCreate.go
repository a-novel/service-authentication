package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/core"
)

type CredentialsCreateService interface {
	Exec(ctx context.Context, request *core.CredentialsCreateRequest) (*core.Token, error)
}

type CredentialsCreateRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	ShortCode string `json:"shortCode"`
}

type CredentialsCreate struct {
	service CredentialsCreateService
	logger  logging.Log
}

func NewCredentialsCreate(service CredentialsCreateService, logger logging.Log) *CredentialsCreate {
	return &CredentialsCreate{service: service, logger: logger}
}

func (handler *CredentialsCreate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.CredentialsCreate")
	defer span.End()

	var request CredentialsCreateRequest

	err := httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	res, err := handler.service.Exec(ctx, &core.CredentialsCreateRequest{
		Email:     request.Email,
		Password:  request.Password,
		ShortCode: request.ShortCode,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
			core.ErrCredentialsCreateAlreadyExists: http.StatusConflict,
			core.ErrShortCodeNotFound:              http.StatusForbidden,
			core.ErrShortCodeConsumeInvalid:        http.StatusForbidden,
			core.ErrInvalidRequest:                 http.StatusUnprocessableEntity,
		}, err)

		return
	}

	httpf.SendJSONStatus(ctx, w, span, http.StatusOK, loadToken(res))
}
