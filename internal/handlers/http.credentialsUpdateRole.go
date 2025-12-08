package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/handlers/middlewares"
	"github.com/a-novel/service-authentication/v2/internal/services"
)

type CredentialsUpdateRoleService interface {
	Exec(ctx context.Context, request *services.CredentialsUpdateRoleRequest) (*services.Credentials, error)
}

type CredentialsUpdateRoleRequest struct {
	UserID uuid.UUID `json:"userID"`
	Role   string    `json:"role"`
}

type CredentialsUpdateRole struct {
	service CredentialsUpdateRoleService
}

func NewCredentialsUpdateRole(service CredentialsUpdateRoleService) *CredentialsUpdateRole {
	return &CredentialsUpdateRole{service: service}
}

func (handler *CredentialsUpdateRole) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "handler.CredentialsUpdateRole")
	defer span.End()

	decoder := json.NewDecoder(r.Body)

	var request CredentialsUpdateRoleRequest

	err := decoder.Decode(&request)
	if err != nil {
		httpf.HandleError(ctx, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	claims, err := middlewares.MustGetClaimsContext(ctx)
	if err != nil {
		httpf.HandleError(ctx, w, span, nil, err)

		return
	}

	res, err := handler.service.Exec(ctx, &services.CredentialsUpdateRoleRequest{
		TargetUserID:  request.UserID,
		CurrentUserID: lo.FromPtr(claims.UserID),
		Role:          request.Role,
	})
	if err != nil {
		httpf.HandleError(ctx, w, span, httpf.ErrMap{
			dao.ErrCredentialsUpdateRoleNotFound:               http.StatusNotFound,
			services.ErrCredentialsUpdateRoleToHigher:          http.StatusForbidden,
			services.ErrCredentialsUpdateRoleDowngradeSuperior: http.StatusForbidden,
			services.ErrCredentialsUpdateRoleSelfUpdate:        http.StatusForbidden,
			services.ErrInvalidRequest:                         http.StatusUnprocessableEntity,
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
