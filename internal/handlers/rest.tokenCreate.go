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

// RestTokenCreateService is the core operation PUT /v2/session delegates to.
type RestTokenCreateService interface {
	Exec(ctx context.Context, request *core.TokenCreateRequest) (*core.Token, error)
}

// tokenCreateErrors maps a refused login to its status. An unknown email and a wrong
// password both answer 401, so a caller cannot probe which addresses are registered.
var tokenCreateErrors = httpf.ErrMap{
	core.ErrCredentialsByEmailNotFound: http.StatusUnauthorized,
	lib.ErrInvalidPassword:             http.StatusUnauthorized,
	core.ErrInvalidRequest:             http.StatusUnprocessableEntity,
}

// RestTokenCreateRequest is the JSON body of PUT /v2/session.
type RestTokenCreateRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RestTokenCreate serves PUT /v2/session, which signs a user in.
type RestTokenCreate struct {
	service RestTokenCreateService
	logger  logging.Log
}

func NewRestTokenCreate(service RestTokenCreateService, logger logging.Log) *RestTokenCreate {
	return &RestTokenCreate{service: service, logger: logger}
}

func (handler *RestTokenCreate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.TokenCreate")
	defer span.End()

	var request RestTokenCreateRequest

	err := httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.TokenCreateRequest{
		Email:    request.Email,
		Password: request.Password,
	}

	res, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, tokenCreateErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadToken(res))
}
