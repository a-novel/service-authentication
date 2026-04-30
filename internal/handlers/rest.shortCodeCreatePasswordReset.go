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

type ShortCodeCreatePasswordResetService interface {
	Exec(ctx context.Context, request *services.ShortCodeCreatePasswordResetRequest) (*services.ShortCode, error)
}

type ShortCodeCreatePasswordResetRequest struct {
	Email string `json:"email"`
	Lang  string `json:"lang"`
}

type ShortCodeCreatePasswordReset struct {
	service ShortCodeCreatePasswordResetService
	logger  logging.Log
}

func NewShortCodeCreatePasswordReset(
	service ShortCodeCreatePasswordResetService, logger logging.Log,
) *ShortCodeCreatePasswordReset {
	return &ShortCodeCreatePasswordReset{service: service, logger: logger}
}

func (handler *ShortCodeCreatePasswordReset) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "handler.ShortCodeCreatePasswordReset")
	defer span.End()

	decoder := json.NewDecoder(r.Body)

	var request ShortCodeCreatePasswordResetRequest

	err := decoder.Decode(&request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	_, err = handler.service.Exec(ctx, &services.ShortCodeCreatePasswordResetRequest{
		Email: request.Email,
		Lang:  request.Lang,
	})
	if err != nil {
		// Silently succeed if email not found to prevent email enumeration.
		// The user won't receive an email, but they won't know if the email was registered.
		if !errors.Is(err, dao.ErrCredentialsSelectByEmailNotFound) {
			httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
				services.ErrInvalidRequest: http.StatusUnprocessableEntity,
			}, err)

			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
}
