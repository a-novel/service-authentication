package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/a-novel/golib/httpf"
	"github.com/a-novel/golib/otel"

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
}

func NewShortCodeCreateRegister(service ShortCodeCreateRegisterService) *ShortCodeCreateRegister {
	return &ShortCodeCreateRegister{service: service}
}

func (handler *ShortCodeCreateRegister) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "handler.ShortCodeCreateRegister")
	defer span.End()

	decoder := json.NewDecoder(r.Body)

	var request ShortCodeCreateRegisterRequest

	err := decoder.Decode(&request)
	if err != nil {
		httpf.HandleError(ctx, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	_, err = handler.service.Exec(ctx, &services.ShortCodeCreateRegisterRequest{
		Email: request.Email,
		Lang:  request.Lang,
	})
	if err != nil {
		httpf.HandleError(ctx, w, span, httpf.ErrMap{
			services.ErrInvalidRequest: http.StatusUnprocessableEntity,
		}, err)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
}
