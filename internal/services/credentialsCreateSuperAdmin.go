package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/lib"
)

type CredentialsCreateSuperAdminRepository interface {
	Exec(ctx context.Context, request *dao.CredentialsInsertRequest) (*dao.Credentials, error)
}
type CredentialsCreateSuperAdminRepositorySelect interface {
	Exec(ctx context.Context, request *dao.CredentialsSelectByEmailRequest) (*dao.Credentials, error)
}
type CredentialsCreateSuperAdminRepositoryUpdatePassword interface {
	Exec(ctx context.Context, request *dao.CredentialsUpdatePasswordRequest) (*dao.Credentials, error)
}
type CredentialsCreateSuperAdminRepositoryUpdateRole interface {
	Exec(ctx context.Context, request *dao.CredentialsUpdateRoleRequest) (*dao.Credentials, error)
}

type CredentialsCreateSuperAdminRequest struct {
	Email    string `validate:"required,email"`
	Password string `validate:"required,max=1024"`
}

type CredentialsCreateSuperAdmin struct {
	repository               CredentialsCreateSuperAdminRepository
	repositorySelect         CredentialsCreateSuperAdminRepositorySelect
	repositoryUpdatePassword CredentialsCreateSuperAdminRepositoryUpdatePassword
	repositoryUpdateRole     CredentialsCreateSuperAdminRepositoryUpdateRole
}

func NewCredentialsCreateSuperAdmin(
	repository CredentialsCreateSuperAdminRepository,
	repositorySelect CredentialsCreateSuperAdminRepositorySelect,
	repositoryUpdatePassword CredentialsCreateSuperAdminRepositoryUpdatePassword,
	repositoryUpdateRole CredentialsCreateSuperAdminRepositoryUpdateRole,
) *CredentialsCreateSuperAdmin {
	return &CredentialsCreateSuperAdmin{
		repository:               repository,
		repositorySelect:         repositorySelect,
		repositoryUpdatePassword: repositoryUpdatePassword,
		repositoryUpdateRole:     repositoryUpdateRole,
	}
}

func (service *CredentialsCreateSuperAdmin) Exec(
	ctx context.Context, request *CredentialsCreateSuperAdminRequest,
) (*Credentials, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.CredentialsCreateSuperAdmin")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	encryptedPassword, err := lib.GenerateScrypt(request.Password, lib.ScryptParamsDefault)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("encrypt password: %w", err))
	}

	var credentials *dao.Credentials

	err = postgres.RunInTx(ctx, nil, func(ctx context.Context, tx bun.IDB) error {
		now := time.Now()

		// Check if user exists.
		credentials, err = service.repositorySelect.Exec(ctx, &dao.CredentialsSelectByEmailRequest{
			Email: request.Email,
		})
		if errors.Is(err, dao.ErrCredentialsSelectByEmailNotFound) {
			// Create credentials.
			credentials, err = service.repository.Exec(ctx, &dao.CredentialsInsertRequest{
				ID:       uuid.New(),
				Email:    request.Email,
				Password: encryptedPassword,
				Now:      now,
				Role:     config.RoleSuperAdmin,
			})
			if err != nil {
				return fmt.Errorf("insert credentials: %w", err)
			}

			return nil
		}
		// If the error was not found, we already returned, so an error here is unexpected.
		if err != nil {
			return err
		}

		credentials, err = service.repositoryUpdatePassword.Exec(ctx, &dao.CredentialsUpdatePasswordRequest{
			ID:       credentials.ID,
			Password: encryptedPassword,
			Now:      now,
		})
		if err != nil {
			return fmt.Errorf("update password: %w", err)
		}

		if credentials.Role != config.RoleSuperAdmin {
			credentials, err = service.repositoryUpdateRole.Exec(ctx, &dao.CredentialsUpdateRoleRequest{
				ID:   credentials.ID,
				Role: config.RoleSuperAdmin,
				Now:  now,
			})
			if err != nil {
				return fmt.Errorf("update role: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return otel.ReportSuccess(span, &Credentials{
		ID:        credentials.ID,
		Email:     credentials.Email,
		Role:      credentials.Role,
		CreatedAt: credentials.CreatedAt,
		UpdatedAt: credentials.UpdatedAt,
	}), nil
}
