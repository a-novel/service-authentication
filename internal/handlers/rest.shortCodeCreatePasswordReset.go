package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/core"
)

// RestShortCodeCreatePasswordResetService is the core operation PUT /v2/short-code/update-password delegates to.
type RestShortCodeCreatePasswordResetService interface {
	Exec(ctx context.Context, request *core.ShortCodeCreatePasswordResetRequest) (*core.ShortCode, error)
}

// RestShortCodeCreatePasswordResetRequest is the JSON body of PUT /v2/short-code/update-password.
type RestShortCodeCreatePasswordResetRequest struct {
	Email string `json:"email"`
	Lang  string `json:"lang"`
}

// RestShortCodeCreatePasswordReset serves PUT /v2/short-code/update-password, which emails a password reset code.
type RestShortCodeCreatePasswordReset struct {
	service RestShortCodeCreatePasswordResetService
	logger  logging.Log
}

func NewRestShortCodeCreatePasswordReset(
	service RestShortCodeCreatePasswordResetService, logger logging.Log,
) *RestShortCodeCreatePasswordReset {
	return &RestShortCodeCreatePasswordReset{service: service, logger: logger}
}

func (handler *RestShortCodeCreatePasswordReset) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ShortCodeCreatePasswordReset")
	defer span.End()

	var request RestShortCodeCreatePasswordResetRequest

	err := httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.ShortCodeCreatePasswordResetRequest{
		Email: request.Email,
		Lang:  request.Lang,
	}

	_, err = handler.service.Exec(ctx, serviceRequest)
	// Silently succeed when the email is unknown, so a caller cannot probe which addresses are registered.
	if err != nil && !errors.Is(err, core.ErrCredentialsByEmailNotFound) {
		errMap := httpf.ErrMap{
			core.ErrInvalidRequest:          http.StatusUnprocessableEntity,
			core.ErrMailDeliveryUnavailable: http.StatusServiceUnavailable,
		}
		httpf.HandleError(ctx, handler.logger, w, span, errMap, err)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
}
