package core_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
)

func TestCredentialsReconcileRole(t *testing.T) {
	t.Parallel()

	request := &core.CredentialsReconcileRoleRequest{
		Email: "operator@provider.com",
		Lang:  config.LangEN,
		Role:  config.RoleAdmin,
	}
	credentialsID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	errFoo := errors.New("foo")

	t.Run("Success/Unchanged", func(t *testing.T) {
		t.Parallel()

		selectCredentials, updateRole, selectShortCode, register, service := newCredentialsReconcileRoleService(t)
		selectCredentials.EXPECT().
			Exec(mock.Anything, &dao.CredentialsSelectByEmailRequest{Email: request.Email}).
			Return(&dao.Credentials{ID: credentialsID, Email: request.Email, Role: request.Role}, nil)

		result, err := service.Exec(t.Context(), request)
		require.NoError(t, err)
		require.Equal(t, core.CredentialsReconcileRoleUnchanged, result.Outcome)

		selectCredentials.AssertExpectations(t)
		updateRole.AssertExpectations(t)
		selectShortCode.AssertExpectations(t)
		register.AssertExpectations(t)
	})

	t.Run("Success/Updated", func(t *testing.T) {
		t.Parallel()

		selectCredentials, updateRole, selectShortCode, register, service := newCredentialsReconcileRoleService(t)
		selectCredentials.EXPECT().
			Exec(mock.Anything, &dao.CredentialsSelectByEmailRequest{Email: request.Email}).
			Return(&dao.Credentials{ID: credentialsID, Email: request.Email, Role: config.RoleSuperAdmin}, nil)
		updateRole.EXPECT().
			Exec(mock.Anything, mock.MatchedBy(func(update *dao.CredentialsUpdateRoleRequest) bool {
				return update.ID == credentialsID &&
					update.Role == request.Role &&
					time.Since(update.Now) < time.Minute
			})).
			Return(&dao.Credentials{ID: credentialsID, Email: request.Email, Role: request.Role}, nil)

		result, err := service.Exec(t.Context(), request)
		require.NoError(t, err)
		require.Equal(t, core.CredentialsReconcileRoleUpdated, result.Outcome)

		selectCredentials.AssertExpectations(t)
		updateRole.AssertExpectations(t)
		selectShortCode.AssertExpectations(t)
		register.AssertExpectations(t)
	})

	t.Run("Success/MatchingRegistrationPending", func(t *testing.T) {
		t.Parallel()

		selectCredentials, updateRole, selectShortCode, register, service := newCredentialsReconcileRoleService(t)
		selectCredentials.EXPECT().
			Exec(mock.Anything, &dao.CredentialsSelectByEmailRequest{Email: request.Email}).
			Return(nil, dao.ErrCredentialsSelectByEmailNotFound)
		selectShortCode.EXPECT().
			Exec(mock.Anything, &dao.ShortCodeSelectRequest{
				Usage:  core.ShortCodeUsageRegister,
				Target: request.Email,
			}).
			Return(&dao.ShortCode{Data: []byte(`{"role":"auth:admin"}`)}, nil)

		result, err := service.Exec(t.Context(), request)
		require.NoError(t, err)
		require.Equal(t, core.CredentialsReconcileRoleRegistrationPending, result.Outcome)

		selectCredentials.AssertExpectations(t)
		updateRole.AssertExpectations(t)
		selectShortCode.AssertExpectations(t)
		register.AssertExpectations(t)
	})

	t.Run("Success/RegistrationCreated", func(t *testing.T) {
		t.Parallel()

		selectCredentials, updateRole, selectShortCode, register, service := newCredentialsReconcileRoleService(t)
		selectCredentials.EXPECT().
			Exec(mock.Anything, &dao.CredentialsSelectByEmailRequest{Email: request.Email}).
			Return(nil, dao.ErrCredentialsSelectByEmailNotFound)
		selectShortCode.EXPECT().
			Exec(mock.Anything, &dao.ShortCodeSelectRequest{
				Usage:  core.ShortCodeUsageRegister,
				Target: request.Email,
			}).
			Return(nil, dao.ErrShortCodeSelectNotFound)
		register.EXPECT().
			Exec(mock.Anything, &core.ShortCodeCreateRegisterRequest{
				Email: request.Email,
				Lang:  request.Lang,
				Role:  request.Role,
			}).
			Return(&core.ShortCode{}, nil)

		result, err := service.Exec(t.Context(), request)
		require.NoError(t, err)
		require.Equal(t, core.CredentialsReconcileRoleRegistrationCreated, result.Outcome)

		selectCredentials.AssertExpectations(t)
		updateRole.AssertExpectations(t)
		selectShortCode.AssertExpectations(t)
		register.AssertExpectations(t)
	})

	t.Run("Success/ReplacesDifferentRegistrationRole", func(t *testing.T) {
		t.Parallel()

		selectCredentials, updateRole, selectShortCode, register, service := newCredentialsReconcileRoleService(t)
		selectCredentials.EXPECT().
			Exec(mock.Anything, &dao.CredentialsSelectByEmailRequest{Email: request.Email}).
			Return(nil, dao.ErrCredentialsSelectByEmailNotFound)
		selectShortCode.EXPECT().
			Exec(mock.Anything, mock.Anything).
			Return(&dao.ShortCode{Data: []byte(`{"role":"auth:user"}`)}, nil)
		register.EXPECT().
			Exec(mock.Anything, &core.ShortCodeCreateRegisterRequest{
				Email: request.Email,
				Lang:  request.Lang,
				Role:  request.Role,
			}).
			Return(&core.ShortCode{}, nil)

		result, err := service.Exec(t.Context(), request)
		require.NoError(t, err)
		require.Equal(t, core.CredentialsReconcileRoleRegistrationCreated, result.Outcome)

		selectCredentials.AssertExpectations(t)
		updateRole.AssertExpectations(t)
		selectShortCode.AssertExpectations(t)
		register.AssertExpectations(t)
	})

	t.Run("Success/ReconcilesRegistrationRace", func(t *testing.T) {
		t.Parallel()

		selectCredentials, updateRole, selectShortCode, register, service := newCredentialsReconcileRoleService(t)
		selectCredentials.EXPECT().
			Exec(mock.Anything, &dao.CredentialsSelectByEmailRequest{Email: request.Email}).
			Return(nil, dao.ErrCredentialsSelectByEmailNotFound).
			Once()
		selectCredentials.EXPECT().
			Exec(mock.Anything, &dao.CredentialsSelectByEmailRequest{Email: request.Email}).
			Return(&dao.Credentials{ID: credentialsID, Email: request.Email, Role: config.RoleUser}, nil).
			Once()
		selectShortCode.EXPECT().
			Exec(mock.Anything, mock.Anything).
			Return(nil, dao.ErrShortCodeSelectNotFound)
		register.EXPECT().
			Exec(mock.Anything, mock.Anything).
			Return(nil, core.ErrCredentialsCreateAlreadyExists)
		updateRole.EXPECT().
			Exec(mock.Anything, mock.MatchedBy(func(update *dao.CredentialsUpdateRoleRequest) bool {
				return update.ID == credentialsID && update.Role == request.Role
			})).
			Return(&dao.Credentials{ID: credentialsID, Email: request.Email, Role: request.Role}, nil)

		result, err := service.Exec(t.Context(), request)
		require.NoError(t, err)
		require.Equal(t, core.CredentialsReconcileRoleUpdated, result.Outcome)

		selectCredentials.AssertExpectations(t)
		updateRole.AssertExpectations(t)
		selectShortCode.AssertExpectations(t)
		register.AssertExpectations(t)
	})

	t.Run("Error/InvalidRole", func(t *testing.T) {
		t.Parallel()

		selectCredentials, updateRole, selectShortCode, register, service := newCredentialsReconcileRoleService(t)
		invalidRequest := *request
		invalidRequest.Role = "auth:unknown"

		result, err := service.Exec(t.Context(), &invalidRequest)
		require.ErrorIs(t, err, core.ErrInvalidRequest)
		require.Nil(t, result)

		selectCredentials.AssertExpectations(t)
		updateRole.AssertExpectations(t)
		selectShortCode.AssertExpectations(t)
		register.AssertExpectations(t)
	})

	t.Run("Error/SelectCredentials", func(t *testing.T) {
		t.Parallel()

		selectCredentials, updateRole, selectShortCode, register, service := newCredentialsReconcileRoleService(t)
		selectCredentials.EXPECT().
			Exec(mock.Anything, mock.Anything).
			Return(nil, errFoo)

		result, err := service.Exec(t.Context(), request)
		require.ErrorIs(t, err, errFoo)
		require.Nil(t, result)

		selectCredentials.AssertExpectations(t)
		updateRole.AssertExpectations(t)
		selectShortCode.AssertExpectations(t)
		register.AssertExpectations(t)
	})

	t.Run("Error/UpdateRole", func(t *testing.T) {
		t.Parallel()

		selectCredentials, updateRole, selectShortCode, register, service := newCredentialsReconcileRoleService(t)
		selectCredentials.EXPECT().
			Exec(mock.Anything, mock.Anything).
			Return(&dao.Credentials{ID: credentialsID, Email: request.Email, Role: config.RoleUser}, nil)
		updateRole.EXPECT().
			Exec(mock.Anything, mock.Anything).
			Return(nil, errFoo)

		result, err := service.Exec(t.Context(), request)
		require.ErrorIs(t, err, errFoo)
		require.Nil(t, result)

		selectCredentials.AssertExpectations(t)
		updateRole.AssertExpectations(t)
		selectShortCode.AssertExpectations(t)
		register.AssertExpectations(t)
	})

	t.Run("Error/SelectRegistration", func(t *testing.T) {
		t.Parallel()

		selectCredentials, updateRole, selectShortCode, register, service := newCredentialsReconcileRoleService(t)
		selectCredentials.EXPECT().
			Exec(mock.Anything, mock.Anything).
			Return(nil, dao.ErrCredentialsSelectByEmailNotFound)
		selectShortCode.EXPECT().
			Exec(mock.Anything, mock.Anything).
			Return(nil, errFoo)

		result, err := service.Exec(t.Context(), request)
		require.ErrorIs(t, err, errFoo)
		require.Nil(t, result)

		selectCredentials.AssertExpectations(t)
		updateRole.AssertExpectations(t)
		selectShortCode.AssertExpectations(t)
		register.AssertExpectations(t)
	})

	t.Run("Error/CreateRegistration", func(t *testing.T) {
		t.Parallel()

		selectCredentials, updateRole, selectShortCode, register, service := newCredentialsReconcileRoleService(t)
		selectCredentials.EXPECT().
			Exec(mock.Anything, mock.Anything).
			Return(nil, dao.ErrCredentialsSelectByEmailNotFound)
		selectShortCode.EXPECT().
			Exec(mock.Anything, mock.Anything).
			Return(nil, dao.ErrShortCodeSelectNotFound)
		register.EXPECT().
			Exec(mock.Anything, mock.Anything).
			Return(nil, errFoo)

		result, err := service.Exec(t.Context(), request)
		require.ErrorIs(t, err, errFoo)
		require.Nil(t, result)

		selectCredentials.AssertExpectations(t)
		updateRole.AssertExpectations(t)
		selectShortCode.AssertExpectations(t)
		register.AssertExpectations(t)
	})
}

func newCredentialsReconcileRoleService(t *testing.T) (
	*coremocks.MockCredentialsReconcileRoleDaoSelect,
	*coremocks.MockCredentialsReconcileRoleDaoUpdate,
	*coremocks.MockCredentialsReconcileRoleDaoShortCodeSelect,
	*coremocks.MockCredentialsReconcileRoleServiceRegister,
	*core.CredentialsReconcileRole,
) {
	t.Helper()

	selectCredentials := coremocks.NewMockCredentialsReconcileRoleDaoSelect(t)
	updateRole := coremocks.NewMockCredentialsReconcileRoleDaoUpdate(t)
	selectShortCode := coremocks.NewMockCredentialsReconcileRoleDaoShortCodeSelect(t)
	register := coremocks.NewMockCredentialsReconcileRoleServiceRegister(t)

	return selectCredentials, updateRole, selectShortCode, register, core.NewCredentialsReconcileRole(
		selectCredentials,
		updateRole,
		selectShortCode,
		register,
	)
}
