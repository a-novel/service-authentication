package handlers

import (
	"context"
	"net/http"

	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/handlers/middlewares"
	"github.com/a-novel/service-authentication/v2/internal/lib"
)

// RestCredentialsUpdatePasswordService is the core operation PATCH /v2/credentials/password delegates to.
type RestCredentialsUpdatePasswordService interface {
	Exec(ctx context.Context, request *core.CredentialsUpdatePasswordRequest) (*core.Credentials, error)
}

// credentialsUpdatePasswordErrors maps a refused password change to its status.
var credentialsUpdatePasswordErrors = httpf.ErrMap{
	core.ErrCredentialsUpdatePasswordNotFound: http.StatusNotFound,
	lib.ErrInvalidPassword:                    http.StatusForbidden,
	core.ErrInvalidRequest:                    http.StatusUnprocessableEntity,
}

// RestCredentialsUpdatePasswordRequest is the JSON body of PATCH /v2/credentials/password.
type RestCredentialsUpdatePasswordRequest struct {
	Password        string `json:"password"`
	CurrentPassword string `json:"currentPassword"`
}

// RestCredentialsUpdatePassword serves PATCH /v2/credentials/password, which changes the caller's password.
type RestCredentialsUpdatePassword struct {
	service RestCredentialsUpdatePasswordService
	logger  logging.Log
}

func NewRestCredentialsUpdatePassword(
	service RestCredentialsUpdatePasswordService, logger logging.Log,
) *RestCredentialsUpdatePassword {
	return &RestCredentialsUpdatePassword{service: service, logger: logger}
}

func (handler *RestCredentialsUpdatePassword) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.CredentialsUpdatePassword")
	defer span.End()

	var request RestCredentialsUpdatePasswordRequest

	err := httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	claims, err := middlewares.MustGetClaimsContext(ctx)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, nil, err)

		return
	}

	serviceRequest := &core.CredentialsUpdatePasswordRequest{
		Password:        request.Password,
		CurrentPassword: request.CurrentPassword,
		UserID:          lo.FromPtr(claims.UserID),
	}

	res, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, credentialsUpdatePasswordErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadCredentials(res))
}
