package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/dao"
)

// CredentialsReconcileRoleDaoSelect loads credentials by email.
type CredentialsReconcileRoleDaoSelect interface {
	Exec(ctx context.Context, request *dao.CredentialsSelectByEmailRequest) (*dao.Credentials, error)
}

// CredentialsReconcileRoleDaoUpdate assigns the requested role to existing credentials.
type CredentialsReconcileRoleDaoUpdate interface {
	Exec(ctx context.Context, request *dao.CredentialsUpdateRoleRequest) (*dao.Credentials, error)
}

// CredentialsReconcileRoleDaoShortCodeSelect loads an active registration code.
type CredentialsReconcileRoleDaoShortCodeSelect interface {
	Exec(ctx context.Context, request *dao.ShortCodeSelectRequest) (*dao.ShortCode, error)
}

// CredentialsReconcileRoleServiceRegister starts registration with the requested role.
type CredentialsReconcileRoleServiceRegister interface {
	Exec(ctx context.Context, request *ShortCodeCreateRegisterRequest) (*ShortCode, error)
}

// CredentialsReconcileRoleRequest identifies the account or registration to reconcile.
type CredentialsReconcileRoleRequest struct {
	Email string `validate:"required,email,max=1024"`
	Lang  string `validate:"required,langs"`
	Role  string `validate:"required,role"`
}

// CredentialsReconcileRoleOutcome describes the state transition performed by reconciliation.
type CredentialsReconcileRoleOutcome string

const (
	// CredentialsReconcileRoleUnchanged means existing credentials already used the requested role.
	CredentialsReconcileRoleUnchanged CredentialsReconcileRoleOutcome = "unchanged"
	// CredentialsReconcileRoleUpdated means existing credentials were assigned the requested role.
	CredentialsReconcileRoleUpdated CredentialsReconcileRoleOutcome = "updated"
	// CredentialsReconcileRoleRegistrationPending means a matching active registration already exists.
	CredentialsReconcileRoleRegistrationPending CredentialsReconcileRoleOutcome = "registration-pending"
	// CredentialsReconcileRoleRegistrationCreated means a registration email was scheduled.
	CredentialsReconcileRoleRegistrationCreated CredentialsReconcileRoleOutcome = "registration-created"
)

// CredentialsReconcileRoleResult reports how the requested state was reached.
type CredentialsReconcileRoleResult struct {
	Outcome CredentialsReconcileRoleOutcome
}

// CredentialsReconcileRole makes an account or pending registration converge on an exact role.
type CredentialsReconcileRole struct {
	selectCredentials CredentialsReconcileRoleDaoSelect
	updateRole        CredentialsReconcileRoleDaoUpdate
	selectShortCode   CredentialsReconcileRoleDaoShortCodeSelect
	register          CredentialsReconcileRoleServiceRegister
}

// NewCredentialsReconcileRole wires account-role reconciliation to persistence and registration.
func NewCredentialsReconcileRole(
	selectCredentials CredentialsReconcileRoleDaoSelect,
	updateRole CredentialsReconcileRoleDaoUpdate,
	selectShortCode CredentialsReconcileRoleDaoShortCodeSelect,
	register CredentialsReconcileRoleServiceRegister,
) *CredentialsReconcileRole {
	return &CredentialsReconcileRole{
		selectCredentials: selectCredentials,
		updateRole:        updateRole,
		selectShortCode:   selectShortCode,
		register:          register,
	}
}

// Exec assigns the requested role to existing credentials or creates a role-bearing
// registration invitation when the email is not registered.
func (service *CredentialsReconcileRole) Exec(
	ctx context.Context,
	request *CredentialsReconcileRoleRequest,
) (*CredentialsReconcileRoleResult, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.CredentialsReconcileRole")
	defer span.End()

	span.SetAttributes(
		attribute.String("credentials.email", request.Email),
		attribute.String("credentials.role", request.Role),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	credentials, err := service.selectCredentials.Exec(ctx, &dao.CredentialsSelectByEmailRequest{
		Email: request.Email,
	})
	if err == nil {
		result, reconcileErr := service.reconcileExisting(ctx, credentials, request.Role)
		if reconcileErr != nil {
			return nil, otel.ReportError(span, reconcileErr)
		}

		if result.Outcome == CredentialsReconcileRoleUnchanged {
			span.SetAttributes(attribute.Bool("noop", true))
		}

		return otel.ReportSuccess(span, result), nil
	}

	if !errors.Is(err, dao.ErrCredentialsSelectByEmailNotFound) {
		return nil, otel.ReportError(span, fmt.Errorf("select credentials: %w", err))
	}

	shortCode, err := service.selectShortCode.Exec(ctx, &dao.ShortCodeSelectRequest{
		Usage:  ShortCodeUsageRegister,
		Target: request.Email,
	})
	if err == nil {
		pendingRole, roleErr := credentialsCreateRole(shortCode.Data)
		if roleErr == nil && pendingRole == request.Role {
			span.SetAttributes(attribute.Bool("noop", true))

			return otel.ReportSuccess(span, &CredentialsReconcileRoleResult{
				Outcome: CredentialsReconcileRoleRegistrationPending,
			}), nil
		}
	} else if !errors.Is(err, dao.ErrShortCodeSelectNotFound) {
		return nil, otel.ReportError(span, fmt.Errorf("select registration code: %w", err))
	}

	_, err = service.register.Exec(ctx, &ShortCodeCreateRegisterRequest{
		Email: request.Email,
		Lang:  request.Lang,
		Role:  request.Role,
	})
	if err != nil {
		if errors.Is(err, ErrCredentialsCreateAlreadyExists) {
			credentials, selectErr := service.selectCredentials.Exec(ctx, &dao.CredentialsSelectByEmailRequest{
				Email: request.Email,
			})
			if selectErr != nil {
				return nil, otel.ReportError(span, fmt.Errorf(
					"select credentials after registration race: %w",
					errors.Join(err, selectErr),
				))
			}

			result, reconcileErr := service.reconcileExisting(ctx, credentials, request.Role)
			if reconcileErr != nil {
				return nil, otel.ReportError(span, reconcileErr)
			}

			if result.Outcome == CredentialsReconcileRoleUnchanged {
				span.SetAttributes(attribute.Bool("noop", true))
			}

			return otel.ReportSuccess(span, result), nil
		}

		return nil, otel.ReportError(span, fmt.Errorf("create registration: %w", err))
	}

	return otel.ReportSuccess(span, &CredentialsReconcileRoleResult{
		Outcome: CredentialsReconcileRoleRegistrationCreated,
	}), nil
}

func (service *CredentialsReconcileRole) reconcileExisting(
	ctx context.Context,
	credentials *dao.Credentials,
	role string,
) (*CredentialsReconcileRoleResult, error) {
	if credentials.Role == role {
		return &CredentialsReconcileRoleResult{
			Outcome: CredentialsReconcileRoleUnchanged,
		}, nil
	}

	_, err := service.updateRole.Exec(ctx, &dao.CredentialsUpdateRoleRequest{
		ID:   credentials.ID,
		Role: role,
		Now:  time.Now(),
	})
	if err != nil {
		return nil, fmt.Errorf("update credentials role: %w", err)
	}

	return &CredentialsReconcileRoleResult{
		Outcome: CredentialsReconcileRoleUpdated,
	}, nil
}
