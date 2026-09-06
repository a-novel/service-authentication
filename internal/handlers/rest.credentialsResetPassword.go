package handlers

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/core"
)

type CredentialsResetPasswordService interface {
	Exec(ctx context.Context, request *core.CredentialsUpdatePasswordRequest) (*core.Credentials, error)
}

type CredentialsResetPasswordRequest struct {
	Password  string    `json:"password"`
	ShortCode string    `json:"shortCode"`
	UserID    uuid.UUID `json:"userID"`
}

type CredentialsResetPassword struct {
	service CredentialsResetPasswordService
	logger  logging.Log
}

func NewCredentialsResetPassword(
	service CredentialsResetPasswordService, logger logging.Log,
) *CredentialsResetPassword {
	return &CredentialsResetPassword{service: service, logger: logger}
}

func (handler *CredentialsResetPassword) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.CredentialsResetPassword")
	defer span.End()

	var request CredentialsResetPasswordRequest

	err := httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	res, err := handler.service.Exec(ctx, &core.CredentialsUpdatePasswordRequest{
		Password:  request.Password,
		ShortCode: request.ShortCode,
		UserID:    request.UserID,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
			core.ErrCredentialsUpdatePasswordNotFound: http.StatusForbidden,
			core.ErrShortCodeNotFound:                 http.StatusForbidden,
			core.ErrShortCodeConsumeInvalid:           http.StatusForbidden,
			core.ErrShortCodeConsumeExpired:           http.StatusForbidden,
			core.ErrInvalidRequest:                    http.StatusUnprocessableEntity,
		}, err)

		return
	}

	httpf.SendJSONStatus(ctx, w, span, http.StatusOK, loadCredentials(res))
}
