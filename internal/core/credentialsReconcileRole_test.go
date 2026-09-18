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
	invalidRequest := *request
	invalidRequest.Role = "auth:unknown"

	credentialsID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	errFoo := errors.New("foo")

	type selectCredentialsMock struct {
		resp *dao.Credentials
		err  error
	}

	type updateRoleMock struct {
		resp *dao.Credentials
		err  error
	}

	type selectShortCodeMock struct {
		resp *dao.ShortCode
		err  error
	}

	type registerMock struct {
		resp *core.ShortCode
		err  error
	}

	testCases := []struct {
		name string

		request *core.CredentialsReconcileRoleRequest

		selectCredentialsMock *selectCredentialsMock
		updateRoleMock        *updateRoleMock
		selectShortCodeMock   *selectShortCodeMock
		registerMock          *registerMock

		expect    *core.CredentialsReconcileRoleResult
		expectErr error
	}{
		{
			name:    "Success/Unchanged",
			request: request,

			selectCredentialsMock: &selectCredentialsMock{
				resp: &dao.Credentials{
					ID:    credentialsID,
					Email: request.Email,
					Role:  request.Role,
				},
			},

			expect: &core.CredentialsReconcileRoleResult{
				Outcome: core.CredentialsReconcileRoleUnchanged,
			},
		},
		{
			name:    "Success/Updated",
			request: request,

			selectCredentialsMock: &selectCredentialsMock{
				resp: &dao.Credentials{
					ID:    credentialsID,
					Email: request.Email,
					Role:  config.RoleSuperAdmin,
				},
			},
			updateRoleMock: &updateRoleMock{
				resp: &dao.Credentials{
					ID:    credentialsID,
					Email: request.Email,
					Role:  request.Role,
				},
			},

			expect: &core.CredentialsReconcileRoleResult{
				Outcome: core.CredentialsReconcileRoleUpdated,
			},
		},
		{
			name:    "Success/MatchingRegistrationPending",
			request: request,

			selectCredentialsMock: &selectCredentialsMock{
				err: dao.ErrCredentialsSelectByEmailNotFound,
			},
			selectShortCodeMock: &selectShortCodeMock{
				resp: &dao.ShortCode{Data: []byte(`{"role":"auth:admin"}`)},
			},

			expect: &core.CredentialsReconcileRoleResult{
				Outcome: core.CredentialsReconcileRoleRegistrationPending,
			},
		},
		{
			name:    "Success/RegistrationCreated",
			request: request,

			selectCredentialsMock: &selectCredentialsMock{
				err: dao.ErrCredentialsSelectByEmailNotFound,
			},
			selectShortCodeMock: &selectShortCodeMock{
				err: dao.ErrShortCodeSelectNotFound,
			},
			registerMock: &registerMock{
				resp: &core.ShortCode{},
			},

			expect: &core.CredentialsReconcileRoleResult{
				Outcome: core.CredentialsReconcileRoleRegistrationCreated,
			},
		},
		{
			name:    "Success/ReplacesDifferentRegistrationRole",
			request: request,

			selectCredentialsMock: &selectCredentialsMock{
				err: dao.ErrCredentialsSelectByEmailNotFound,
			},
			selectShortCodeMock: &selectShortCodeMock{
				resp: &dao.ShortCode{Data: []byte(`{"role":"auth:user"}`)},
			},
			registerMock: &registerMock{
				resp: &core.ShortCode{},
			},

			expect: &core.CredentialsReconcileRoleResult{
				Outcome: core.CredentialsReconcileRoleRegistrationCreated,
			},
		},
		{
			name:      "Error/InvalidRole",
			request:   &invalidRequest,
			expectErr: core.ErrInvalidRequest,
		},
		{
			name:    "Error/SelectCredentials",
			request: request,

			selectCredentialsMock: &selectCredentialsMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name:    "Error/UpdateRole",
			request: request,

			selectCredentialsMock: &selectCredentialsMock{
				resp: &dao.Credentials{
					ID:    credentialsID,
					Email: request.Email,
					Role:  config.RoleUser,
				},
			},
			updateRoleMock: &updateRoleMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name:    "Error/SelectRegistration",
			request: request,

			selectCredentialsMock: &selectCredentialsMock{
				err: dao.ErrCredentialsSelectByEmailNotFound,
			},
			selectShortCodeMock: &selectShortCodeMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name:    "Error/CreateRegistration",
			request: request,

			selectCredentialsMock: &selectCredentialsMock{
				err: dao.ErrCredentialsSelectByEmailNotFound,
			},
			selectShortCodeMock: &selectShortCodeMock{
				err: dao.ErrShortCodeSelectNotFound,
			},
			registerMock: &registerMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name:    "Error/RegistrationRace",
			request: request,

			selectCredentialsMock: &selectCredentialsMock{
				err: dao.ErrCredentialsSelectByEmailNotFound,
			},
			selectShortCodeMock: &selectShortCodeMock{
				err: dao.ErrShortCodeSelectNotFound,
			},
			registerMock: &registerMock{
				err: core.ErrCredentialsCreateAlreadyExists,
			},

			expectErr: core.ErrCredentialsCreateAlreadyExists,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			selectCredentials := coremocks.NewMockCredentialsReconcileRoleDaoSelect(t)
			updateRole := coremocks.NewMockCredentialsReconcileRoleDaoUpdate(t)
			selectShortCode := coremocks.NewMockCredentialsReconcileRoleDaoShortCodeSelect(t)
			register := coremocks.NewMockCredentialsReconcileRoleServiceRegister(t)

			if testCase.selectCredentialsMock != nil {
				selectCredentials.EXPECT().
					Exec(mock.Anything, &dao.CredentialsSelectByEmailRequest{
						Email: testCase.request.Email,
					}).
					Return(
						testCase.selectCredentialsMock.resp,
						testCase.selectCredentialsMock.err,
					)
			}

			if testCase.updateRoleMock != nil {
				updateRole.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(update *dao.CredentialsUpdateRoleRequest) bool {
						return update.ID == credentialsID &&
							update.Role == testCase.request.Role &&
							time.Since(update.Now) < time.Minute
					})).
					Return(testCase.updateRoleMock.resp, testCase.updateRoleMock.err)
			}

			if testCase.selectShortCodeMock != nil {
				selectShortCode.EXPECT().
					Exec(mock.Anything, &dao.ShortCodeSelectRequest{
						Usage:  core.ShortCodeUsageRegister,
						Target: testCase.request.Email,
					}).
					Return(
						testCase.selectShortCodeMock.resp,
						testCase.selectShortCodeMock.err,
					)
			}

			if testCase.registerMock != nil {
				register.EXPECT().
					Exec(mock.Anything, &core.ShortCodeCreateRegisterRequest{
						Email: testCase.request.Email,
						Lang:  testCase.request.Lang,
						Role:  testCase.request.Role,
					}).
					Return(testCase.registerMock.resp, testCase.registerMock.err)
			}

			service := core.NewCredentialsReconcileRole(
				selectCredentials,
				updateRole,
				selectShortCode,
				register,
			)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			selectCredentials.AssertExpectations(t)
			updateRole.AssertExpectations(t)
			selectShortCode.AssertExpectations(t)
			register.AssertExpectations(t)
		})
	}
}
