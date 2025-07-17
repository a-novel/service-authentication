package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/models"
)

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

type UpdateRoleRequest struct {
	TargetUserID  uuid.UUID
	CurrentUserID uuid.UUID
	Role          models.CredentialsRole
}

type UpdateRoleService struct {
	source UpdateRoleSource
}

func NewUpdateRoleService(source UpdateRoleSource) *UpdateRoleService {
	return &UpdateRoleService{source: source}
}

func (service *UpdateRoleService) UpdateRole(
	ctx context.Context, request UpdateRoleRequest,
) (*models.User, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.UpdateRole")
	defer span.End()

	span.SetAttributes(
		attribute.String("request.targetUserID", request.TargetUserID.String()),
		attribute.String("request.currentUserID", request.CurrentUserID.String()),
		attribute.String("request.role", request.Role.String()),
	)

	// No self-update allowed.
	if request.CurrentUserID == request.TargetUserID {
		return nil, otel.ReportError(span, ErrSelfRoleUpdate)
	}

	newTargetRoleImportance := models.KnownCredentialsRolesWithImportance[request.Role]
	span.SetAttributes(attribute.Int("newTargetRoleImportance", newTargetRoleImportance.Int()))

	// Role importance start at 1.
	if newTargetRoleImportance == models.CredentialRoleImportanceUnknown {
		return nil, otel.ReportError(span, fmt.Errorf("%w: %s", ErrUnknownRole, request.Role))
	}

	// Retrieve the target user credentials.
	targetCredentials, err := service.source.SelectCredentials(ctx, request.TargetUserID)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select target credentials: %w", err))
	}

	span.SetAttributes(attribute.String("targetCredentials.email", targetCredentials.Email))

	// Retrieve the current user credentials.
	currentCredentials, err := service.source.SelectCredentials(ctx, request.CurrentUserID)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select current user credentials: %w", err))
	}

	span.SetAttributes(attribute.String("currentCredentials.email", currentCredentials.Email))

	targetRoleIImportance := models.KnownCredentialsRolesWithImportance[targetCredentials.Role]
	currentRoleIImportance := models.KnownCredentialsRolesWithImportance[currentCredentials.Role]

	span.SetAttributes(
		attribute.Int("targetRoleImportance", targetRoleIImportance.Int()),
		attribute.Int("currentRoleImportance", currentRoleIImportance.Int()),
	)

	// User can only upgrade users up to its own role.
	if newTargetRoleImportance >= targetRoleIImportance && newTargetRoleImportance > currentRoleIImportance {
		return nil, otel.ReportError(span,
			fmt.Errorf("%w: upgrade from %s to %s", ErrUpdateToHigherRole, currentCredentials.Role, request.Role),
		)
	}

	// User can only downgrade users from a lower role.
	if newTargetRoleImportance <= targetRoleIImportance && targetRoleIImportance >= currentRoleIImportance {
		return nil, otel.ReportError(span,
			fmt.Errorf("%w: downgrade from %s to %s", ErrMustDowngradeLowerRole, currentCredentials.Role, request.Role),
		)
	}

	// If new role is equal to the current role, return the target credentials (noop).
	if newTargetRoleImportance == targetRoleIImportance {
		span.SetAttributes(attribute.Bool("noop", true))

		return otel.ReportSuccess(span, &models.User{
			ID:        targetCredentials.ID,
			Email:     targetCredentials.Email,
			Role:      targetCredentials.Role,
			CreatedAt: targetCredentials.CreatedAt,
			UpdatedAt: targetCredentials.UpdatedAt,
		}), nil
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
		return nil, otel.ReportError(span, err)
	}

	return otel.ReportSuccess(span, &models.User{
		ID:        updatedCredentials.ID,
		Email:     updatedCredentials.Email,
		Role:      updatedCredentials.Role,
		CreatedAt: updatedCredentials.CreatedAt,
		UpdatedAt: updatedCredentials.UpdatedAt,
	}), nil
}
