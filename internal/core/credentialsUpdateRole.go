package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/dao"
)

var (
	// ErrCredentialsUpdateRoleSelfUpdate is returned by [CredentialsUpdateRole.Exec]
	// when the actor and the target are the same user. Self-promotion or
	// self-demotion is never allowed, even for super-admins.
	ErrCredentialsUpdateRoleSelfUpdate = errors.New("user is not allowed to update its own role")
	// ErrCredentialsUpdateRoleDowngradeSuperior is returned by
	// [CredentialsUpdateRole.Exec] when the actor tries to demote a user whose
	// current role is equal to or higher than the actor's own.
	ErrCredentialsUpdateRoleDowngradeSuperior = errors.New("user can only downgrade users from a lower role")
	// ErrCredentialsUpdateRoleToHigher is returned by [CredentialsUpdateRole.Exec]
	// when the actor tries to promote a user to a role above the actor's own.
	ErrCredentialsUpdateRoleToHigher = errors.New(
		"user is not allowed to upgrade users to a higher role than its own",
	)
)

type CredentialsUpdateRoleDao interface {
	Exec(ctx context.Context, request *dao.CredentialsUpdateRoleRequest) (*dao.Credentials, error)
}

type CredentialsUpdateRoleDaoCredentialsSelect interface {
	Exec(ctx context.Context, request *dao.CredentialsSelectRequest) (*dao.Credentials, error)
}

type CredentialsUpdateRoleRequest struct {
	TargetUserID  uuid.UUID
	CurrentUserID uuid.UUID
	Role          string `validate:"required,role"`
}

// CredentialsUpdateRole changes a target user's role on behalf of an acting user,
// enforcing the role hierarchy: an actor cannot change its own role, promote anyone
// above its own rank, or demote a user whose rank is at or above its own.
type CredentialsUpdateRole struct {
	dao                  CredentialsUpdateRoleDao
	daoCredentialsSelect CredentialsUpdateRoleDaoCredentialsSelect
}

func NewCredentialsUpdateRole(
	dao CredentialsUpdateRoleDao,
	daoCredentialsSelect CredentialsUpdateRoleDaoCredentialsSelect,
) *CredentialsUpdateRole {
	return &CredentialsUpdateRole{
		dao:                  dao,
		daoCredentialsSelect: daoCredentialsSelect,
	}
}

func (service *CredentialsUpdateRole) Exec(
	ctx context.Context, request *CredentialsUpdateRoleRequest,
) (*Credentials, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.CredentialsUpdateRole")
	defer span.End()

	span.SetAttributes(
		attribute.String("target.id", request.TargetUserID.String()),
		attribute.String("actor.id", request.CurrentUserID.String()),
		attribute.String("target.role", request.Role),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	if request.CurrentUserID == request.TargetUserID {
		return nil, otel.ReportError(span, ErrCredentialsUpdateRoleSelfUpdate)
	}

	// request.Role passed validate:"role", so it is known; the lookup cannot fail.
	newTargetRoleImportance, err := config.PermissionsConfigDefault.Priority(request.Role)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	span.SetAttributes(attribute.Int("newTargetRoleImportance", newTargetRoleImportance))

	targetCredentials, err := service.daoCredentialsSelect.Exec(ctx, &dao.CredentialsSelectRequest{
		ID: request.TargetUserID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select target credentials: %w", err))
	}

	span.SetAttributes(attribute.String("targetCredentials.email", targetCredentials.Email))

	currentCredentials, err := service.daoCredentialsSelect.Exec(ctx, &dao.CredentialsSelectRequest{
		ID: request.CurrentUserID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select current user credentials: %w", err))
	}

	span.SetAttributes(attribute.String("currentCredentials.email", currentCredentials.Email))

	// These two roles come from the database, not the request, so they carry no
	// validation. A stored role the config no longer knows is an error here, not a
	// silent priority 0 that would let the rank guards below compare against a rank
	// the account does not have.
	targetRoleIImportance, err := config.PermissionsConfigDefault.Priority(targetCredentials.Role)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("rank target role: %w", err))
	}

	currentRoleIImportance, err := config.PermissionsConfigDefault.Priority(currentCredentials.Role)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("rank current user role: %w", err))
	}

	span.SetAttributes(
		attribute.Int("targetRoleImportance", targetRoleIImportance),
		attribute.Int("currentRoleImportance", currentRoleIImportance),
	)

	// User can only upgrade users up to its own role.
	if newTargetRoleImportance >= targetRoleIImportance && newTargetRoleImportance > currentRoleIImportance {
		return nil, otel.ReportError(span, fmt.Errorf(
			"%w: upgrade from %s to %s",
			ErrCredentialsUpdateRoleToHigher, currentCredentials.Role, request.Role,
		))
	}

	// User can only downgrade users from a lower role.
	if newTargetRoleImportance <= targetRoleIImportance && targetRoleIImportance >= currentRoleIImportance {
		return nil, otel.ReportError(span, fmt.Errorf(
			"%w: downgrade from %s to %s",
			ErrCredentialsUpdateRoleDowngradeSuperior, currentCredentials.Role, request.Role,
		))
	}

	// Same rank as the target already holds: nothing to update.
	if newTargetRoleImportance == targetRoleIImportance {
		span.SetAttributes(attribute.Bool("noop", true))

		return otel.ReportSuccess(span, &Credentials{
			ID:        targetCredentials.ID,
			Email:     targetCredentials.Email,
			Role:      targetCredentials.Role,
			CreatedAt: targetCredentials.CreatedAt,
			UpdatedAt: targetCredentials.UpdatedAt,
		}), nil
	}

	updatedCredentials, err := service.dao.Exec(
		ctx,
		&dao.CredentialsUpdateRoleRequest{
			ID:   request.TargetUserID,
			Role: request.Role,
			Now:  time.Now(),
		},
	)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	return otel.ReportSuccess(span, &Credentials{
		ID:        updatedCredentials.ID,
		Email:     updatedCredentials.Email,
		Role:      updatedCredentials.Role,
		CreatedAt: updatedCredentials.CreatedAt,
		UpdatedAt: updatedCredentials.UpdatedAt,
	}), nil
}
