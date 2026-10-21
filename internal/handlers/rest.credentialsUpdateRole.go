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

// RestCredentialsUpdateRoleService is the core operation PATCH /v2/credentials/role delegates to.
type RestCredentialsUpdateRoleService interface {
	Exec(ctx context.Context, request *core.CredentialsUpdateRoleRequest) (*core.Credentials, error)
}

// credentialsUpdateRoleErrors maps a refused role change to its status.
var credentialsUpdateRoleErrors = httpf.ErrMap{
	core.ErrCredentialsUpdateRoleNotFound: http.StatusNotFound,
	// The select raises this when the target or actor credentials are missing.
	core.ErrCredentialsGetNotFound:                 http.StatusNotFound,
	core.ErrCredentialsUpdateRoleToHigher:          http.StatusForbidden,
	core.ErrCredentialsUpdateRoleDowngradeSuperior: http.StatusForbidden,
	core.ErrCredentialsUpdateRoleSelfUpdate:        http.StatusForbidden,
	core.ErrInvalidRequest:                         http.StatusUnprocessableEntity,
}

// RestCredentialsUpdateRoleRequest is the JSON body of PATCH /v2/credentials/role.
type RestCredentialsUpdateRoleRequest struct {
	UserID uuid.UUID `json:"userID"`
	Role   string    `json:"role"`
}

// RestCredentialsUpdateRole serves PATCH /v2/credentials/role, which changes an account's role.
type RestCredentialsUpdateRole struct {
	service RestCredentialsUpdateRoleService
	logger  logging.Log
}

func NewRestCredentialsUpdateRole(
	service RestCredentialsUpdateRoleService, logger logging.Log,
) *RestCredentialsUpdateRole {
	return &RestCredentialsUpdateRole{service: service, logger: logger}
}

func (handler *RestCredentialsUpdateRole) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.CredentialsUpdateRole")
	defer span.End()

	var request RestCredentialsUpdateRoleRequest

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

	serviceRequest := &core.CredentialsUpdateRoleRequest{
		TargetUserID:  request.UserID,
		CurrentUserID: lo.FromPtr(claims.UserID),
		Role:          request.Role,
	}

	res, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, credentialsUpdateRoleErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadCredentials(res))
}
