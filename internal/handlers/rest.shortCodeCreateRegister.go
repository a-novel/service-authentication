package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/core"
)

// RestShortCodeCreateRegisterService is the core operation PUT /v2/short-code/register delegates to.
type RestShortCodeCreateRegisterService interface {
	Exec(ctx context.Context, request *core.ShortCodeCreateRegisterRequest) (*core.ShortCode, error)
}

// shortCodeCreateRegisterErrors maps a failed invitation to its status.
var shortCodeCreateRegisterErrors = httpf.ErrMap{
	core.ErrInvalidRequest:          http.StatusUnprocessableEntity,
	core.ErrMailDeliveryUnavailable: http.StatusServiceUnavailable,
}

// RestShortCodeCreateRegisterRequest is the JSON body of PUT /v2/short-code/register.
type RestShortCodeCreateRegisterRequest struct {
	Email string `json:"email"`
	Lang  string `json:"lang"`
}

// RestShortCodeCreateRegister serves PUT /v2/short-code/register, which emails a registration invitation.
type RestShortCodeCreateRegister struct {
	service RestShortCodeCreateRegisterService
	logger  logging.Log
}

func NewRestShortCodeCreateRegister(
	service RestShortCodeCreateRegisterService, logger logging.Log,
) *RestShortCodeCreateRegister {
	return &RestShortCodeCreateRegister{service: service, logger: logger}
}

func (handler *RestShortCodeCreateRegister) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ShortCodeCreateRegister")
	defer span.End()

	var request RestShortCodeCreateRegisterRequest

	err := httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.ShortCodeCreateRegisterRequest{
		Email: request.Email,
		Lang:  request.Lang,
		Role:  authconfig.RoleUser,
	}

	_, err = handler.service.Exec(ctx, serviceRequest)
	// Silently succeed when the email already exists, so a caller cannot probe which addresses are registered.
	if err != nil && !errors.Is(err, core.ErrCredentialsCreateAlreadyExists) {
		httpf.HandleError(ctx, handler.logger, w, span, shortCodeCreateRegisterErrors, err)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
}
