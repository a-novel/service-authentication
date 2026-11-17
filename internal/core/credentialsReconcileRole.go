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
	Exec(ctx context.Context, request *ShortCodeCreateRegisterRequest) error
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
	daoSelect          CredentialsReconcileRoleDaoSelect
	daoUpdate          CredentialsReconcileRoleDaoUpdate
	daoShortCodeSelect CredentialsReconcileRoleDaoShortCodeSelect
	serviceRegister    CredentialsReconcileRoleServiceRegister
}

// NewCredentialsReconcileRole wires account-role reconciliation to persistence and registration.
func NewCredentialsReconcileRole(
	daoSelect CredentialsReconcileRoleDaoSelect,
	daoUpdate CredentialsReconcileRoleDaoUpdate,
	daoShortCodeSelect CredentialsReconcileRoleDaoShortCodeSelect,
	serviceRegister CredentialsReconcileRoleServiceRegister,
) *CredentialsReconcileRole {
	return &CredentialsReconcileRole{
		daoSelect:          daoSelect,
		daoUpdate:          daoUpdate,
		daoShortCodeSelect: daoShortCodeSelect,
		serviceRegister:    serviceRegister,
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

	credentials, err := service.daoSelect.Exec(ctx, &dao.CredentialsSelectByEmailRequest{Email: request.Email})
	if err != nil && !errors.Is(err, dao.ErrCredentialsNotFound) {
		return nil, otel.ReportError(span, fmt.Errorf("select credentials: %w", err))
	}

	// An existing account takes the role in place. A missing one continues to registration.
	if err == nil {
		result, err := service.reconcileExisting(ctx, credentials, request.Role)
		if err != nil {
			return nil, otel.ReportError(span, err)
		}

		if result.Outcome == CredentialsReconcileRoleUnchanged {
			span.SetAttributes(attribute.Bool("credentials.role_unchanged", true))
		}

		return result, nil
	}

	shortCodeRequest := &dao.ShortCodeSelectRequest{
		Usage:  ShortCodeUsageRegister,
		Target: request.Email,
	}

	shortCode, err := service.daoShortCodeSelect.Exec(ctx, shortCodeRequest)
	if err != nil && !errors.Is(err, dao.ErrShortCodeNotFound) {
		return nil, otel.ReportError(span, fmt.Errorf("select registration code: %w", err))
	}

	if err == nil && registrationGrants(shortCode, request.Role) {
		span.SetAttributes(attribute.Bool("credentials.role_unchanged", true))

		return &CredentialsReconcileRoleResult{Outcome: CredentialsReconcileRoleRegistrationPending}, nil
	}

	registerRequest := &ShortCodeCreateRegisterRequest{
		Email: request.Email,
		Lang:  request.Lang,
		Role:  request.Role,
	}

	err = service.serviceRegister.Exec(ctx, registerRequest)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("create registration: %w", err))
	}

	return &CredentialsReconcileRoleResult{Outcome: CredentialsReconcileRoleRegistrationCreated}, nil
}

func (service *CredentialsReconcileRole) reconcileExisting(
	ctx context.Context,
	credentials *dao.Credentials,
	role string,
) (*CredentialsReconcileRoleResult, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.CredentialsReconcileRole(reconcileExisting)")
	defer span.End()

	if credentials.Role == role {
		return &CredentialsReconcileRoleResult{Outcome: CredentialsReconcileRoleUnchanged}, nil
	}

	daoRequest := &dao.CredentialsUpdateRoleRequest{
		ID:   credentials.ID,
		Role: role,
		Now:  time.Now(),
	}

	_, err := service.daoUpdate.Exec(ctx, daoRequest)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("update credentials role: %w", err))
	}

	return &CredentialsReconcileRoleResult{Outcome: CredentialsReconcileRoleUpdated}, nil
}

// registrationGrants reports whether a pending registration code carries role. A code whose data is
// malformed or names an unknown role grants nothing, so a new registration replaces it.
func registrationGrants(shortCode *dao.ShortCode, role string) bool {
	pendingRole, err := computeRole(shortCode.Data)

	return err == nil && pendingRole == role
}
