package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/transaction"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/lib"
)

type CredentialsCreateSuperAdminDao interface {
	Exec(ctx context.Context, request *dao.CredentialsInsertRequest) (*dao.Credentials, error)
}
type CredentialsCreateSuperAdminDaoSelect interface {
	Exec(ctx context.Context, request *dao.CredentialsSelectByEmailRequest) (*dao.Credentials, error)
}
type CredentialsCreateSuperAdminDaoUpdatePassword interface {
	Exec(ctx context.Context, request *dao.CredentialsUpdatePasswordRequest) (*dao.Credentials, error)
}
type CredentialsCreateSuperAdminDaoUpdateRole interface {
	Exec(ctx context.Context, request *dao.CredentialsUpdateRoleRequest) (*dao.Credentials, error)
}

// CredentialsCreateSuperAdminRequest carries the email and password of the
// super-admin account to provision.
type CredentialsCreateSuperAdminRequest struct {
	Email    string `validate:"required,email,max=1024"`
	Password string `validate:"required,min=4,max=1024"`
}

// CredentialsCreateSuperAdmin idempotently provisions a super-admin account for
// bootstrap. Given an email and password it creates the account when absent, and
// otherwise resets that account's password and raises its role to super-admin.
type CredentialsCreateSuperAdmin struct {
	dao               CredentialsCreateSuperAdminDao
	daoSelect         CredentialsCreateSuperAdminDaoSelect
	daoUpdatePassword CredentialsCreateSuperAdminDaoUpdatePassword
	daoUpdateRole     CredentialsCreateSuperAdminDaoUpdateRole
	transactor        transaction.Transactor
}

func NewCredentialsCreateSuperAdmin(
	dao CredentialsCreateSuperAdminDao,
	daoSelect CredentialsCreateSuperAdminDaoSelect,
	daoUpdatePassword CredentialsCreateSuperAdminDaoUpdatePassword,
	daoUpdateRole CredentialsCreateSuperAdminDaoUpdateRole,
	transactor transaction.Transactor,
) *CredentialsCreateSuperAdmin {
	return &CredentialsCreateSuperAdmin{
		dao:               dao,
		daoSelect:         daoSelect,
		daoUpdatePassword: daoUpdatePassword,
		daoUpdateRole:     daoUpdateRole,
		transactor:        transactor,
	}
}

// Exec provisions the super-admin account described by the request, running the
// lookup and the create-or-update in one transaction. The password is hashed with
// Argon2id before storage. An existing account keeps its identity and creation
// time; only its password and, when needed, its role are updated.
func (service *CredentialsCreateSuperAdmin) Exec(
	ctx context.Context, request *CredentialsCreateSuperAdminRequest,
) (*Credentials, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.CredentialsCreateSuperAdmin")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	encryptedPassword, err := lib.GenerateArgon2(request.Password, lib.Argon2ParamsDefault)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("encrypt password: %w", err))
	}

	var credentials *dao.Credentials

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		now := time.Now()

		// Check if user exists.
		credentials, err = service.daoSelect.Exec(ctx, &dao.CredentialsSelectByEmailRequest{
			Email: request.Email,
		})
		if errors.Is(err, dao.ErrCredentialsSelectByEmailNotFound) {
			// Create credentials.
			credentials, err = service.dao.Exec(ctx, &dao.CredentialsInsertRequest{
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
		// The not-found case returned above; any error remaining here is a real lookup failure.
		if err != nil {
			return err
		}

		credentials, err = service.daoUpdatePassword.Exec(ctx, &dao.CredentialsUpdatePasswordRequest{
			ID:       credentials.ID,
			Password: encryptedPassword,
			Now:      now,
		})
		if err != nil {
			return fmt.Errorf("update password: %w", err)
		}

		if credentials.Role != config.RoleSuperAdmin {
			credentials, err = service.daoUpdateRole.Exec(ctx, &dao.CredentialsUpdateRoleRequest{
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
