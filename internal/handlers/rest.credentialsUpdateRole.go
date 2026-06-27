package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/handlers/middlewares"
)

type CredentialsUpdateRoleService interface {
	Exec(ctx context.Context, request *core.CredentialsUpdateRoleRequest) (*core.Credentials, error)
}

type CredentialsUpdateRoleRequest struct {
	UserID uuid.UUID `json:"userID"`
	Role   string    `json:"role"`
}

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

	decoder := json.NewDecoder(r.Body)

	var request CredentialsUpdateRoleRequest

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

	res, err := handler.service.Exec(ctx, &core.CredentialsUpdateRoleRequest{
		TargetUserID:  request.UserID,
		CurrentUserID: lo.FromPtr(claims.UserID),
		Role:          request.Role,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
			dao.ErrCredentialsUpdateRoleNotFound: http.StatusNotFound,
			// The target (or actor) credentials don't exist — surfaced by the select, not the update.
			dao.ErrCredentialsSelectNotFound:               http.StatusNotFound,
			core.ErrCredentialsUpdateRoleToHigher:          http.StatusForbidden,
			core.ErrCredentialsUpdateRoleDowngradeSuperior: http.StatusForbidden,
			core.ErrCredentialsUpdateRoleSelfUpdate:        http.StatusForbidden,
			core.ErrInvalidRequest:                         http.StatusUnprocessableEntity,
		}, err)

		return
	}

	httpf.SendJSON(ctx, w, span, loadCredentials(res))
}
