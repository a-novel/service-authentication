package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/core"
)

// RestCredentialsCreateService is the core operation PUT /v2/credentials delegates to.
type RestCredentialsCreateService interface {
	Exec(ctx context.Context, request *core.CredentialsCreateRequest) (*core.Token, error)
}

// RestCredentialsCreateRequest is the JSON body of PUT /v2/credentials.
type RestCredentialsCreateRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	ShortCode string `json:"shortCode"`
}

// RestCredentialsCreate serves PUT /v2/credentials, which creates an account from a registration short code.
type RestCredentialsCreate struct {
	service RestCredentialsCreateService
	logger  logging.Log
}

func NewRestCredentialsCreate(service RestCredentialsCreateService, logger logging.Log) *RestCredentialsCreate {
	return &RestCredentialsCreate{service: service, logger: logger}
}

func (handler *RestCredentialsCreate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.CredentialsCreate")
	defer span.End()

	var request RestCredentialsCreateRequest

	err := httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.CredentialsCreateRequest{
		Email:     request.Email,
		Password:  request.Password,
		ShortCode: request.ShortCode,
	}

	res, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		errMap := httpf.ErrMap{
			core.ErrCredentialsCreateAlreadyExists: http.StatusConflict,
			core.ErrShortCodeNotFound:              http.StatusForbidden,
			core.ErrShortCodeConsumeInvalid:        http.StatusForbidden,
			core.ErrShortCodeConsumeExpired:        http.StatusForbidden,
			core.ErrInvalidRequest:                 http.StatusUnprocessableEntity,
		}
		httpf.HandleError(ctx, handler.logger, w, span, errMap, err)

		return
	}

	httpf.SendJSONStatus(ctx, w, span, http.StatusOK, loadToken(res))
}
