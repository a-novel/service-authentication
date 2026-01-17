package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/services"
)

type CredentialsResetPasswordService interface {
	Exec(ctx context.Context, request *services.CredentialsUpdatePasswordRequest) (*services.Credentials, error)
}

type CredentialsResetPasswordRequest struct {
	Password  string    `json:"password"`
	ShortCode string    `json:"shortCode"`
	UserID    uuid.UUID `json:"userID"`
}

type CredentialsResetPassword struct {
	service CredentialsResetPasswordService
}

func NewCredentialsResetPassword(service CredentialsResetPasswordService) *CredentialsResetPassword {
	return &CredentialsResetPassword{service: service}
}

func (handler *CredentialsResetPassword) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "handler.CredentialsResetPassword")
	defer span.End()

	decoder := json.NewDecoder(r.Body)

	var request CredentialsResetPasswordRequest

	err := decoder.Decode(&request)
	if err != nil {
		httpf.HandleError(ctx, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	res, err := handler.service.Exec(ctx, &services.CredentialsUpdatePasswordRequest{
		Password:  request.Password,
		ShortCode: request.ShortCode,
		UserID:    request.UserID,
	})
	if err != nil {
		httpf.HandleError(ctx, w, span, httpf.ErrMap{
			dao.ErrCredentialsUpdatePasswordNotFound: http.StatusForbidden,
			dao.ErrShortCodeSelectNotFound:           http.StatusForbidden,
			services.ErrShortCodeConsumeInvalid:      http.StatusForbidden,
			services.ErrInvalidRequest:               http.StatusUnprocessableEntity,
		}, err)

		return
	}

	httpf.SendJSON(ctx, w, span, loadCredentials(res))
}
