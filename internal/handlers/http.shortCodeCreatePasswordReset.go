package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/a-novel/golib/httpf"
	"github.com/a-novel/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/services"
)

type ShortCodeCreatePasswordResetService interface {
	Exec(ctx context.Context, request *services.ShortCodeCreatePasswordResetRequest) (*services.ShortCode, error)
}

type ShortCodeCreatePasswordResetRequest struct {
	Email string `json:"email"`
	Lang  string `json:"lang"`
}

type ShortCodeCreatePasswordReset struct {
	service ShortCodeCreatePasswordResetService
}

func NewShortCodeCreatePasswordReset(service ShortCodeCreatePasswordResetService) *ShortCodeCreatePasswordReset {
	return &ShortCodeCreatePasswordReset{service: service}
}

func (handler *ShortCodeCreatePasswordReset) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "handler.ShortCodeCreatePasswordReset")
	defer span.End()

	decoder := json.NewDecoder(r.Body)

	var request ShortCodeCreatePasswordResetRequest

	err := decoder.Decode(&request)
	if err != nil {
		httpf.HandleError(ctx, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	_, err = handler.service.Exec(ctx, &services.ShortCodeCreatePasswordResetRequest{
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
