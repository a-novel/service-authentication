package handlers

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/handlers/middlewares"
)

// CredentialsUpdateRoleService is the core operation PATCH /v2/credentials/role delegates to.
type CredentialsUpdateRoleService interface {
	Exec(ctx context.Context, request *core.CredentialsUpdateRoleRequest) (*core.Credentials, error)
}

// CredentialsUpdateRoleRequest is the JSON body of PATCH /v2/credentials/role.
type CredentialsUpdateRoleRequest struct {
	UserID uuid.UUID `json:"userID"`
	Role   string    `json:"role"`
}

// CredentialsUpdateRole serves PATCH /v2/credentials/role, which changes an account's role.
type CredentialsUpdateRole struct {
	service CredentialsUpdateRoleService
	logger  logging.Log
}

func NewCredentialsUpdateRole(service CredentialsUpdateRoleService, logger logging.Log) *CredentialsUpdateRole {
	return &CredentialsUpdateRole{service: service, logger: logger}
}

func (handler *CredentialsUpdateRole) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.CredentialsUpdateRole")
	defer span.End()

	var request CredentialsUpdateRoleRequest

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

	res, err := handler.service.Exec(ctx, &core.CredentialsUpdateRoleRequest{
		TargetUserID:  request.UserID,
		CurrentUserID: lo.FromPtr(claims.UserID),
		Role:          request.Role,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
			core.ErrCredentialsUpdateRoleNotFound: http.StatusNotFound,
			// The select raises this when the target or actor credentials are missing.
			core.ErrCredentialsGetNotFound:                 http.StatusNotFound,
			core.ErrCredentialsUpdateRoleToHigher:          http.StatusForbidden,
			core.ErrCredentialsUpdateRoleDowngradeSuperior: http.StatusForbidden,
			core.ErrCredentialsUpdateRoleSelfUpdate:        http.StatusForbidden,
			core.ErrInvalidRequest:                         http.StatusUnprocessableEntity,
		}, err)

		return
	}

	httpf.SendJSONStatus(ctx, w, span, http.StatusOK, loadCredentials(res))
}
