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

type ShortCodeCreateRegisterService interface {
	Exec(ctx context.Context, request *core.ShortCodeCreateRegisterRequest) (*core.ShortCode, error)
}

type ShortCodeCreateRegisterRequest struct {
	Email string `json:"email"`
	Lang  string `json:"lang"`
}

type ShortCodeCreateRegister struct {
	service ShortCodeCreateRegisterService
	logger  logging.Log
}

func NewShortCodeCreateRegister(service ShortCodeCreateRegisterService, logger logging.Log) *ShortCodeCreateRegister {
	return &ShortCodeCreateRegister{service: service, logger: logger}
}

func (handler *ShortCodeCreateRegister) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ShortCodeCreateRegister")
	defer span.End()

	var request ShortCodeCreateRegisterRequest

	err := httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	_, err = handler.service.Exec(ctx, &core.ShortCodeCreateRegisterRequest{
		Email: request.Email,
		Lang:  request.Lang,
	})
	if err != nil {
		// Silently succeed when the email already exists, so a caller cannot probe which addresses are registered.
		if !errors.Is(err, core.ErrCredentialsCreateAlreadyExists) {
			httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
				core.ErrInvalidRequest: http.StatusUnprocessableEntity,
			}, err)

			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
}
