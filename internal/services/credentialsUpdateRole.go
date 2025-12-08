package services

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
	ErrCredentialsUpdateRoleSelfUpdate        = errors.New("user is not allowed to update its own role")
	ErrCredentialsUpdateRoleDowngradeSuperior = errors.New("user cam only downgrade users from a lower role")
	ErrCredentialsUpdateRoleToHigher          = errors.New(
		"user is not allowed to upgrade users to a higher role than its own",
	)
)

type CredentialsUpdateRoleRepository interface {
	Exec(ctx context.Context, request *dao.CredentialsUpdateRoleRequest) (*dao.Credentials, error)
}

type CredentialsUpdateRoleRepositoryCredentialsSelect interface {
	Exec(ctx context.Context, request *dao.CredentialsSelectRequest) (*dao.Credentials, error)
}

type CredentialsUpdateRoleRequest struct {
	TargetUserID  uuid.UUID
	CurrentUserID uuid.UUID
	Role          string `validate:"required,role"`
}

type CredentialsUpdateRole struct {
	repository                  CredentialsUpdateRoleRepository
	repositoryCredentialsSelect CredentialsUpdateRoleRepositoryCredentialsSelect
}

func NewCredentialsUpdateRole(
	repository CredentialsUpdateRoleRepository,
	repositoryCredentialsSelect CredentialsUpdateRoleRepositoryCredentialsSelect,
) *CredentialsUpdateRole {
	return &CredentialsUpdateRole{
		repository:                  repository,
		repositoryCredentialsSelect: repositoryCredentialsSelect,
	}
}

func (service *CredentialsUpdateRole) Exec(
	ctx context.Context, request *CredentialsUpdateRoleRequest,
) (*Credentials, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.CredentialsUpdateRole")
	defer span.End()

	span.SetAttributes(
		attribute.String("request.targetUserID", request.TargetUserID.String()),
		attribute.String("request.currentUserID", request.CurrentUserID.String()),
		attribute.String("request.role", request.Role),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	// No self-update allowed.
	if request.CurrentUserID == request.TargetUserID {
		return nil, otel.ReportError(span, ErrCredentialsUpdateRoleSelfUpdate)
	}

	newTargetRoleImportance := config.PermissionsConfigDefault.Roles[request.Role].Priority
	span.SetAttributes(attribute.Int("newTargetRoleImportance", newTargetRoleImportance))

	// Retrieve the target user credentials.
	targetCredentials, err := service.repositoryCredentialsSelect.Exec(ctx, &dao.CredentialsSelectRequest{
		ID: request.TargetUserID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select target credentials: %w", err))
	}

	span.SetAttributes(attribute.String("targetCredentials.email", targetCredentials.Email))

	// Retrieve the current user credentials.
	currentCredentials, err := service.repositoryCredentialsSelect.Exec(ctx, &dao.CredentialsSelectRequest{
		ID: request.CurrentUserID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select current user credentials: %w", err))
	}

	span.SetAttributes(attribute.String("currentCredentials.email", currentCredentials.Email))

	targetRoleIImportance := config.PermissionsConfigDefault.Roles[targetCredentials.Role].Priority
	currentRoleIImportance := config.PermissionsConfigDefault.Roles[currentCredentials.Role].Priority

	span.SetAttributes(
		attribute.Int("targetRoleImportance", targetRoleIImportance),
		attribute.Int("currentRoleImportance", currentRoleIImportance),
	)

	// User can only upgrade users up to its own role.
	if newTargetRoleImportance >= targetRoleIImportance && newTargetRoleImportance > currentRoleIImportance {
		return nil, otel.ReportError(span,
			fmt.Errorf(
				"%w: upgrade from %s to %s",
				ErrCredentialsUpdateRoleToHigher, currentCredentials.Role, request.Role,
			),
		)
	}

	// User can only downgrade users from a lower role.
	if newTargetRoleImportance <= targetRoleIImportance && targetRoleIImportance >= currentRoleIImportance {
		return nil, otel.ReportError(span,
			fmt.Errorf(
				"%w: downgrade from %s to %s",
				ErrCredentialsUpdateRoleDowngradeSuperior, currentCredentials.Role, request.Role,
			),
		)
	}

	// If new role is equal to the current role, return the target credentials (noop).
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

	// Update the role.
	updatedCredentials, err := service.repository.Exec(
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
