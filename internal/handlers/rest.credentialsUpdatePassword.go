package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/handlers/middlewares"
	"github.com/a-novel/service-authentication/v2/internal/lib"
	"github.com/a-novel/service-authentication/v2/internal/services"
)

type CredentialsUpdatePasswordService interface {
	Exec(ctx context.Context, request *services.CredentialsUpdatePasswordRequest) (*services.Credentials, error)
}

type CredentialsUpdatePasswordRequest struct {
	Password        string `json:"password"`
	CurrentPassword string `json:"currentPassword"`
}

type CredentialsUpdatePassword struct {
	service CredentialsUpdatePasswordService
	logger  logging.Log
}

func NewCredentialsUpdatePassword(
	service CredentialsUpdatePasswordService, logger logging.Log,
) *CredentialsUpdatePassword {
	return &CredentialsUpdatePassword{service: service, logger: logger}
}

func (handler *CredentialsUpdatePassword) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "handler.CredentialsUpdatePassword")
	defer span.End()

	decoder := json.NewDecoder(r.Body)

	var request CredentialsUpdatePasswordRequest

	err := decoder.Decode(&request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	claims, err := middlewares.MustGetClaimsContext(ctx)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, nil, err)

		return
	}

	res, err := handler.service.Exec(ctx, &services.CredentialsUpdatePasswordRequest{
		Password:        request.Password,
		CurrentPassword: request.CurrentPassword,
		UserID:          lo.FromPtr(claims.UserID),
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
			dao.ErrCredentialsUpdatePasswordNotFound: http.StatusNotFound,
			lib.ErrInvalidPassword:                   http.StatusForbidden,
			services.ErrInvalidRequest:               http.StatusUnprocessableEntity,
		}, err)

		return
	}

	httpf.SendJSON(ctx, w, span, loadCredentials(res))
}
