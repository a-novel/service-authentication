package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/lib"
	"github.com/a-novel/service-authentication/v2/internal/services"
)

type TokenCreateService interface {
	Exec(ctx context.Context, request *services.TokenCreateRequest) (*services.Token, error)
}

type TokenCreateRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type TokenCreate struct {
	service TokenCreateService
}

func NewTokenCreate(service TokenCreateService) *TokenCreate {
	return &TokenCreate{service: service}
}

func (handler *TokenCreate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "handler.TokenCreate")
	defer span.End()

	decoder := json.NewDecoder(r.Body)

	var request TokenCreateRequest

	err := decoder.Decode(&request)
	if err != nil {
		httpf.HandleError(ctx, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	res, err := handler.service.Exec(ctx, &services.TokenCreateRequest{
		Email:    request.Email,
		Password: request.Password,
	})
	if err != nil {
		// Both "email not found" and "invalid password" return 401 to prevent email enumeration.
		httpf.HandleError(ctx, w, span, httpf.ErrMap{
			dao.ErrCredentialsSelectByEmailNotFound: http.StatusUnauthorized,
			lib.ErrInvalidPassword:                  http.StatusUnauthorized,
			services.ErrInvalidRequest:              http.StatusUnprocessableEntity,
		}, err)

		return
	}

	httpf.SendJSON(ctx, w, span, loadToken(res))
}
