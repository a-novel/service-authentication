package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/services"
)

type ShortCodeCreateRegisterService interface {
	Exec(ctx context.Context, request *services.ShortCodeCreateRegisterRequest) (*services.ShortCode, error)
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
	ctx, span := otel.Tracer().Start(r.Context(), "handler.ShortCodeCreateRegister")
	defer span.End()

	decoder := json.NewDecoder(r.Body)

	var request ShortCodeCreateRegisterRequest

	err := decoder.Decode(&request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	_, err = handler.service.Exec(ctx, &services.ShortCodeCreateRegisterRequest{
		Email: request.Email,
		Lang:  request.Lang,
	})
	if err != nil {
		// Silently succeed if email already exists to prevent email enumeration.
		// The user won't receive an email, but they won't know if the email was registered.
		if !errors.Is(err, dao.ErrCredentialsInsertAlreadyExists) {
			httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
				services.ErrInvalidRequest: http.StatusUnprocessableEntity,
			}, err)

			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
}
