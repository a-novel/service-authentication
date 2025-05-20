package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/models"
)

var ErrUpdateRoleService = errors.New("UpdateRoleService.UpdateRole")

func NewErrUpdateRoleService(err error) error {
	return errors.Join(err, ErrUpdateRoleService)
}

var (
	ErrUpdateToHigherRole     = errors.New("user is not allowed to upgrade users to a higher role than its own")
	ErrMustDowngradeLowerRole = errors.New("user cam only downgrade users from a lower role")
	ErrUnknownRole            = errors.New("unknown role")
	ErrSelfRoleUpdate         = errors.New("user is not allowed to update its own role")
)

type UpdateRoleSource interface {
	UpdateCredentialsRole(
		ctx context.Context, userID uuid.UUID, data dao.UpdateCredentialsRoleData,
	) (*dao.CredentialsEntity, error)
	SelectCredentials(ctx context.Context, id uuid.UUID) (*dao.CredentialsEntity, error)
}

type UpdateRoleRequest struct {
	TargetUserID  uuid.UUID
	CurrentUserID uuid.UUID
	Role          models.CredentialsRole
}

type UpdateRoleService struct {
	source UpdateRoleSource
}

func (service *UpdateRoleService) UpdateRole(
	ctx context.Context, request UpdateRoleRequest,
) (*models.User, error) {
	// No self-update allowed.
	if request.CurrentUserID == request.TargetUserID {
		return nil, NewErrUpdateRoleService(ErrSelfRoleUpdate)
	}

	newTargetRoleImportance := models.KnownCredentialsRolesWithImportance[request.Role]
	// Role importance start at 1.
	if newTargetRoleImportance == models.CredentialRoleImportanceUnknown {
		return nil, NewErrUpdateRoleService(fmt.Errorf("%w: %s", ErrUnknownRole, request.Role))
	}

	// Retrieve the target user credentials.
	targetCredentials, err := service.source.SelectCredentials(ctx, request.TargetUserID)
	if err != nil {
		return nil, NewErrUpdateRoleService(fmt.Errorf("select target credentials: %w", err))
	}

	// Retrieve the current user credentials.
	currentCredentials, err := service.source.SelectCredentials(ctx, request.CurrentUserID)
	if err != nil {
		return nil, NewErrUpdateRoleService(fmt.Errorf("select current user credentials: %w", err))
	}

	targetRoleIImportance := models.KnownCredentialsRolesWithImportance[targetCredentials.Role]
	currentRoleIImportance := models.KnownCredentialsRolesWithImportance[currentCredentials.Role]

	// User can only upgrade users up to its own role.
	if newTargetRoleImportance >= targetRoleIImportance && newTargetRoleImportance > currentRoleIImportance {
		return nil, NewErrUpdateRoleService(
			fmt.Errorf("%w: upgrade from %s to %s", ErrUpdateToHigherRole, currentCredentials.Role, request.Role),
		)
	}

	// User can only downgrade users from a lower role.
	if newTargetRoleImportance <= targetRoleIImportance && targetRoleIImportance >= currentRoleIImportance {
		return nil, NewErrUpdateRoleService(
			fmt.Errorf("%w: downgrade from %s to %s", ErrMustDowngradeLowerRole, currentCredentials.Role, request.Role),
		)
	}

	// If new role is equal to the current role, return the target credentials (noop).
	if newTargetRoleImportance == targetRoleIImportance {
		return &models.User{
			ID:        targetCredentials.ID,
			Email:     targetCredentials.Email,
			Role:      targetCredentials.Role,
			CreatedAt: targetCredentials.CreatedAt,
			UpdatedAt: targetCredentials.UpdatedAt,
		}, nil
	}

	// Update the role.
	updatedCredentials, err := service.source.UpdateCredentialsRole(
		ctx,
		request.TargetUserID, dao.UpdateCredentialsRoleData{
			Role: request.Role,
			Now:  time.Now(),
		},
	)
	if err != nil {
		return nil, NewErrUpdateRoleService(err)
	}

	return &models.User{
		ID:        updatedCredentials.ID,
		Email:     updatedCredentials.Email,
		Role:      updatedCredentials.Role,
		CreatedAt: updatedCredentials.CreatedAt,
		UpdatedAt: updatedCredentials.UpdatedAt,
	}, nil
}

func NewUpdateRoleServiceSource(
	updateRoleDAO *dao.UpdateCredentialsRoleRepository,
	selectCredentialsDAO *dao.SelectCredentialsRepository,
) UpdateRoleSource {
	return &struct {
		*dao.UpdateCredentialsRoleRepository
		*dao.SelectCredentialsRepository
	}{
		UpdateCredentialsRoleRepository: updateRoleDAO,
		SelectCredentialsRepository:     selectCredentialsDAO,
	}
}

func NewUpdateRoleService(source UpdateRoleSource) *UpdateRoleService {
	return &UpdateRoleService{source: source}
}
