package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/dao"
)

type CredentialsUpdateEmailService interface {
	Exec(ctx context.Context, request *core.CredentialsUpdateEmailRequest) (*core.Credentials, error)
}

type CredentialsUpdateEmailRequest struct {
	UserID    uuid.UUID `json:"userID"`
	ShortCode string    `json:"shortCode"`
}

type CredentialsUpdateEmail struct {
	service CredentialsUpdateEmailService
	logger  logging.Log
}

func NewCredentialsUpdateEmail(service CredentialsUpdateEmailService, logger logging.Log) *CredentialsUpdateEmail {
	return &CredentialsUpdateEmail{service: service, logger: logger}
}

func (handler *CredentialsUpdateEmail) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.CredentialsUpdateEmail")
	defer span.End()

	decoder := json.NewDecoder(r.Body)

	var request CredentialsUpdateEmailRequest

	err := decoder.Decode(&request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	res, err := handler.service.Exec(ctx, &core.CredentialsUpdateEmailRequest{
		UserID:    request.UserID,
		ShortCode: request.ShortCode,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
			dao.ErrCredentialsUpdateEmailNotFound:      http.StatusNotFound,
			dao.ErrCredentialsUpdateEmailAlreadyExists: http.StatusConflict,
			dao.ErrShortCodeSelectNotFound:             http.StatusForbidden,
			core.ErrShortCodeConsumeInvalid:            http.StatusForbidden,
			core.ErrInvalidRequest:                     http.StatusUnprocessableEntity,
		}, err)

		return
	}

	httpf.SendJSON(ctx, w, span, loadCredentials(res))
}
