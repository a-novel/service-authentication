package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/handlers/middlewares"
)

// RestShortCodeCreateEmailUpdateService is the core operation PUT /v2/short-code/update-email delegates to.
type RestShortCodeCreateEmailUpdateService interface {
	Exec(ctx context.Context, request *core.ShortCodeCreateEmailUpdateRequest) (*core.ShortCode, error)
}

// RestShortCodeCreateEmailUpdateRequest is the JSON body of PUT /v2/short-code/update-email.
type RestShortCodeCreateEmailUpdateRequest struct {
	Email string `json:"email"`
	Lang  string `json:"lang"`
}

// RestShortCodeCreateEmailUpdate serves PUT /v2/short-code/update-email, which emails a code confirming a new address.
type RestShortCodeCreateEmailUpdate struct {
	service RestShortCodeCreateEmailUpdateService
	logger  logging.Log
}

func NewRestShortCodeCreateEmailUpdate(
	service RestShortCodeCreateEmailUpdateService, logger logging.Log,
) *RestShortCodeCreateEmailUpdate {
	return &RestShortCodeCreateEmailUpdate{service: service, logger: logger}
}

func (handler *RestShortCodeCreateEmailUpdate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ShortCodeCreateEmailUpdate")
	defer span.End()

	var request RestShortCodeCreateEmailUpdateRequest

	err := httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	claims, err := middlewares.MustGetClaimsContext(ctx)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, nil, err)

		return
	}

	serviceRequest := &core.ShortCodeCreateEmailUpdateRequest{
		Email: request.Email,
		Lang:  request.Lang,
		ID:    lo.FromPtr(claims.UserID),
	}

	_, err = handler.service.Exec(ctx, serviceRequest)
	// Silently succeed when the email already exists, so a caller cannot probe which addresses are registered.
	if err != nil && !errors.Is(err, core.ErrCredentialsUpdateEmailAlreadyExists) {
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
