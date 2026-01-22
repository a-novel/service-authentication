package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/handlers/middlewares"
	"github.com/a-novel/service-authentication/v2/internal/services"
)

type ShortCodeCreateEmailUpdateService interface {
	Exec(ctx context.Context, request *services.ShortCodeCreateEmailUpdateRequest) (*services.ShortCode, error)
}

type ShortCodeCreateEmailUpdateRequest struct {
	Email string `json:"email"`
	Lang  string `json:"lang"`
}

type ShortCodeCreateEmailUpdate struct {
	service ShortCodeCreateEmailUpdateService
}

func NewShortCodeCreateEmailUpdate(service ShortCodeCreateEmailUpdateService) *ShortCodeCreateEmailUpdate {
	return &ShortCodeCreateEmailUpdate{service: service}
}

func (handler *ShortCodeCreateEmailUpdate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "handler.ShortCodeCreateEmailUpdate")
	defer span.End()

	decoder := json.NewDecoder(r.Body)

	var request ShortCodeCreateEmailUpdateRequest

	err := decoder.Decode(&request)
	if err != nil {
		httpf.HandleError(ctx, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	claims, err := middlewares.MustGetClaimsContext(ctx)
	if err != nil {
		httpf.HandleError(ctx, w, span, nil, err)

		return
	}

	_, err = handler.service.Exec(ctx, &services.ShortCodeCreateEmailUpdateRequest{
		Email: request.Email,
		Lang:  request.Lang,
		ID:    lo.FromPtr(claims.UserID),
	})
	if err != nil {
		// Silently succeed if email already exists to prevent email enumeration.
		// The user won't receive an email, but they won't know if the email was registered.
		if !errors.Is(err, dao.ErrCredentialsUpdateEmailAlreadyExists) {
			httpf.HandleError(ctx, w, span, httpf.ErrMap{
				services.ErrInvalidRequest: http.StatusUnprocessableEntity,
			}, err)

			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
}
