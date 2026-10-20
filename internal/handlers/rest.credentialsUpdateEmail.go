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

// RestCredentialsUpdateEmailService is the core operation PATCH /v2/credentials/email delegates to.
type RestCredentialsUpdateEmailService interface {
	Exec(ctx context.Context, request *core.CredentialsUpdateEmailRequest) (*core.Credentials, error)
}

// credentialsUpdateEmailErrors maps a refused email change to its status.
var credentialsUpdateEmailErrors = httpf.ErrMap{
	core.ErrCredentialsUpdateEmailNotFound:      http.StatusNotFound,
	core.ErrCredentialsUpdateEmailAlreadyExists: http.StatusConflict,
	core.ErrShortCodeNotFound:                   http.StatusForbidden,
	core.ErrShortCodeConsumeInvalid:             http.StatusForbidden,
	core.ErrShortCodeConsumeExpired:             http.StatusForbidden,
	core.ErrInvalidRequest:                      http.StatusUnprocessableEntity,
}

// RestCredentialsUpdateEmailRequest is the JSON body of PATCH /v2/credentials/email.
type RestCredentialsUpdateEmailRequest struct {
	UserID    uuid.UUID `json:"userID"`
	ShortCode string    `json:"shortCode"`
}

// RestCredentialsUpdateEmail serves PATCH /v2/credentials/email, which applies a confirmed email change.
type RestCredentialsUpdateEmail struct {
	service RestCredentialsUpdateEmailService
	logger  logging.Log
}

func NewRestCredentialsUpdateEmail(
	service RestCredentialsUpdateEmailService, logger logging.Log,
) *RestCredentialsUpdateEmail {
	return &RestCredentialsUpdateEmail{service: service, logger: logger}
}

func (handler *RestCredentialsUpdateEmail) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.CredentialsUpdateEmail")
	defer span.End()

	var request RestCredentialsUpdateEmailRequest

	err := httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.CredentialsUpdateEmailRequest{
		UserID:    request.UserID,
		ShortCode: request.ShortCode,
	}

	res, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, credentialsUpdateEmailErrors, err)

		return
	}

	httpf.SendJSONStatus(ctx, w, span, http.StatusOK, loadCredentials(res))
}
