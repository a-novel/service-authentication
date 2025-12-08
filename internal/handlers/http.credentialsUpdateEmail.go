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

type CredentialsUpdateEmailService interface {
	Exec(ctx context.Context, request *services.CredentialsUpdateEmailRequest) (*services.Credentials, error)
}

type CredentialsUpdateEmailRequest struct {
	UserID    uuid.UUID `json:"userID"`
	ShortCode string    `json:"shortCode"`
}

type CredentialsUpdateEmail struct {
	service CredentialsUpdateEmailService
}

func NewCredentialsUpdateEmail(service CredentialsUpdateEmailService) *CredentialsUpdateEmail {
	return &CredentialsUpdateEmail{service: service}
}

func (handler *CredentialsUpdateEmail) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "handler.CredentialsUpdateEmail")
	defer span.End()

	decoder := json.NewDecoder(r.Body)

	var request CredentialsUpdateEmailRequest

	err := decoder.Decode(&request)
	if err != nil {
		httpf.HandleError(ctx, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	res, err := handler.service.Exec(ctx, &services.CredentialsUpdateEmailRequest{
		UserID:    request.UserID,
		ShortCode: request.ShortCode,
	})
	if err != nil {
		httpf.HandleError(ctx, w, span, httpf.ErrMap{
			dao.ErrCredentialsUpdateEmailNotFound:      http.StatusNotFound,
			dao.ErrCredentialsUpdateEmailAlreadyExists: http.StatusConflict,
			dao.ErrShortCodeSelectNotFound:             http.StatusForbidden,
			services.ErrShortCodeConsumeInvalid:        http.StatusForbidden,
			services.ErrInvalidRequest:                 http.StatusUnprocessableEntity,
		}, err)

		return
	}

	httpf.SendJSON(ctx, w, span, Credentials{
		ID:        res.ID,
		Email:     res.Email,
		Role:      res.Role,
		CreatedAt: res.CreatedAt,
		UpdatedAt: res.UpdatedAt,
	})
}
