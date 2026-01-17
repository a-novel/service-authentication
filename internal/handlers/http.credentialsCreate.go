package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/services"
)

type CredentialsCreateService interface {
	Exec(ctx context.Context, request *services.CredentialsCreateRequest) (*services.Token, error)
}

type CredentialsCreateRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	ShortCode string `json:"shortCode"`
}

type CredentialsCreate struct {
	service CredentialsCreateService
}

func NewCredentialsCreate(service CredentialsCreateService) *CredentialsCreate {
	return &CredentialsCreate{service: service}
}

func (handler *CredentialsCreate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "handler.CredentialsCreate")
	defer span.End()

	decoder := json.NewDecoder(r.Body)

	var request CredentialsCreateRequest

	err := decoder.Decode(&request)
	if err != nil {
		httpf.HandleError(ctx, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	res, err := handler.service.Exec(ctx, &services.CredentialsCreateRequest{
		Email:     request.Email,
		Password:  request.Password,
		ShortCode: request.ShortCode,
	})
	if err != nil {
		httpf.HandleError(ctx, w, span, httpf.ErrMap{
			dao.ErrCredentialsInsertAlreadyExists: http.StatusConflict,
			dao.ErrShortCodeSelectNotFound:        http.StatusForbidden,
			services.ErrShortCodeConsumeInvalid:   http.StatusForbidden,
			services.ErrInvalidRequest:            http.StatusUnprocessableEntity,
		}, err)

		return
	}

	httpf.SendJSON(ctx, w, span, loadToken(res))
}
