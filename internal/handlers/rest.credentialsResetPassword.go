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

// RestCredentialsResetPasswordService is the core operation PUT /v2/credentials/password delegates to.
type RestCredentialsResetPasswordService interface {
	Exec(ctx context.Context, request *core.CredentialsUpdatePasswordRequest) (*core.Credentials, error)
}

// credentialsResetPasswordErrors maps a refused password reset to its status.
var credentialsResetPasswordErrors = httpf.ErrMap{
	core.ErrCredentialsUpdatePasswordNotFound: http.StatusForbidden,
	core.ErrShortCodeNotFound:                 http.StatusForbidden,
	core.ErrShortCodeConsumeInvalid:           http.StatusForbidden,
	core.ErrShortCodeConsumeExpired:           http.StatusForbidden,
	core.ErrInvalidRequest:                    http.StatusUnprocessableEntity,
}

// RestCredentialsResetPasswordRequest is the JSON body of PUT /v2/credentials/password.
type RestCredentialsResetPasswordRequest struct {
	Password  string    `json:"password"`
	ShortCode string    `json:"shortCode"`
	UserID    uuid.UUID `json:"userID"`
}

// RestCredentialsResetPassword serves PUT /v2/credentials/password, which resets a password with a short code.
type RestCredentialsResetPassword struct {
	service RestCredentialsResetPasswordService
	logger  logging.Log
}

func NewRestCredentialsResetPassword(
	service RestCredentialsResetPasswordService, logger logging.Log,
) *RestCredentialsResetPassword {
	return &RestCredentialsResetPassword{service: service, logger: logger}
}

func (handler *RestCredentialsResetPassword) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.CredentialsResetPassword")
	defer span.End()

	var request RestCredentialsResetPasswordRequest

	err := httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.CredentialsUpdatePasswordRequest{
		Password:  request.Password,
		ShortCode: request.ShortCode,
		UserID:    request.UserID,
	}

	res, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, credentialsResetPasswordErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadCredentials(res))
}
