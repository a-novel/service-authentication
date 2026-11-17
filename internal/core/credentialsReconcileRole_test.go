package core_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
)

func TestCredentialsReconcileRole(t *testing.T) {
	t.Parallel()

	request := &core.CredentialsReconcileRoleRequest{
		Email: "operator@provider.com",
		Lang:  authconfig.LangEN,
		Role:  authconfig.RoleAdmin,
	}
	invalidRequest := *request
	invalidRequest.Role = "auth:unknown"

	credentialsID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	errFoo := errors.New("foo")

	type daoSelectMock struct {
		resp *dao.Credentials
		err  error
	}

	type daoUpdateMock struct {
		resp *dao.Credentials
		err  error
	}

	type daoShortCodeSelectMock struct {
		resp *dao.ShortCode
		err  error
	}

	type serviceRegisterMock struct {
		err error
	}

	testCases := []struct {
		name string

		request *core.CredentialsReconcileRoleRequest

		daoSelectMock          *daoSelectMock
		daoUpdateMock          *daoUpdateMock
		daoShortCodeSelectMock *daoShortCodeSelectMock
		serviceRegisterMock    *serviceRegisterMock

		expect    *core.CredentialsReconcileRoleResult
		expectErr error
	}{
		{
			name:    "Success/Unchanged",
			request: request,

			daoSelectMock: &daoSelectMock{
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

			daoSelectMock: &daoSelectMock{
				resp: &dao.Credentials{
					ID:    credentialsID,
					Email: request.Email,
					Role:  authconfig.RoleSuperAdmin,
				},
			},
			daoUpdateMock: &daoUpdateMock{
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

			daoSelectMock: &daoSelectMock{
				err: dao.ErrCredentialsNotFound,
			},
			daoShortCodeSelectMock: &daoShortCodeSelectMock{
				resp: &dao.ShortCode{Data: []byte(`{"role":"auth:admin"}`)},
			},

			expect: &core.CredentialsReconcileRoleResult{
				Outcome: core.CredentialsReconcileRoleRegistrationPending,
			},
		},
		{
			name:    "Success/RegistrationCreated",
			request: request,

			daoSelectMock: &daoSelectMock{
				err: dao.ErrCredentialsNotFound,
			},
			daoShortCodeSelectMock: &daoShortCodeSelectMock{
				err: dao.ErrShortCodeNotFound,
			},
			serviceRegisterMock: &serviceRegisterMock{},

			expect: &core.CredentialsReconcileRoleResult{
				Outcome: core.CredentialsReconcileRoleRegistrationCreated,
			},
		},
		{
			name:    "Success/ReplacesDifferentRegistrationRole",
			request: request,

			daoSelectMock: &daoSelectMock{
				err: dao.ErrCredentialsNotFound,
			},
			daoShortCodeSelectMock: &daoShortCodeSelectMock{
				resp: &dao.ShortCode{Data: []byte(`{"role":"auth:user"}`)},
			},
			serviceRegisterMock: &serviceRegisterMock{},

			expect: &core.CredentialsReconcileRoleResult{
				Outcome: core.CredentialsReconcileRoleRegistrationCreated,
			},
		},
		{
			name:    "Success/ReplacesMalformedRegistration",
			request: request,

			daoSelectMock: &daoSelectMock{
				err: dao.ErrCredentialsNotFound,
			},
			daoShortCodeSelectMock: &daoShortCodeSelectMock{
				resp: &dao.ShortCode{Data: []byte(`{"role":`)},
			},
			serviceRegisterMock: &serviceRegisterMock{},

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

			daoSelectMock: &daoSelectMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name:    "Error/UpdateRole",
			request: request,

			daoSelectMock: &daoSelectMock{
				resp: &dao.Credentials{
					ID:    credentialsID,
					Email: request.Email,
					Role:  authconfig.RoleUser,
				},
			},
			daoUpdateMock: &daoUpdateMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name:    "Error/SelectRegistration",
			request: request,

			daoSelectMock: &daoSelectMock{
				err: dao.ErrCredentialsNotFound,
			},
			daoShortCodeSelectMock: &daoShortCodeSelectMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name:    "Error/CreateRegistration",
			request: request,

			daoSelectMock: &daoSelectMock{
				err: dao.ErrCredentialsNotFound,
			},
			daoShortCodeSelectMock: &daoShortCodeSelectMock{
				err: dao.ErrShortCodeNotFound,
			},
			serviceRegisterMock: &serviceRegisterMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name:    "Error/RegistrationRace",
			request: request,

			daoSelectMock: &daoSelectMock{
				err: dao.ErrCredentialsNotFound,
			},
			daoShortCodeSelectMock: &daoShortCodeSelectMock{
				err: dao.ErrShortCodeNotFound,
			},
			serviceRegisterMock: &serviceRegisterMock{
				err: core.ErrCredentialsAlreadyExists,
			},

			expectErr: core.ErrCredentialsAlreadyExists,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			daoSelect := coremocks.NewMockCredentialsReconcileRoleDaoSelect(t)
			daoUpdate := coremocks.NewMockCredentialsReconcileRoleDaoUpdate(t)
			daoShortCodeSelect := coremocks.NewMockCredentialsReconcileRoleDaoShortCodeSelect(t)
			serviceRegister := coremocks.NewMockCredentialsReconcileRoleServiceRegister(t)

			if testCase.daoSelectMock != nil {
				daoSelect.EXPECT().
					Exec(mock.Anything, &dao.CredentialsSelectByEmailRequest{
						Email: testCase.request.Email,
					}).
					Return(
						testCase.daoSelectMock.resp,
						testCase.daoSelectMock.err,
					)
			}

			if testCase.daoUpdateMock != nil {
				daoUpdate.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(update *dao.CredentialsUpdateRoleRequest) bool {
						return update.ID == credentialsID &&
							update.Role == testCase.request.Role &&
							time.Since(update.Now) < time.Minute
					})).
					Return(testCase.daoUpdateMock.resp, testCase.daoUpdateMock.err)
			}

			if testCase.daoShortCodeSelectMock != nil {
				shortCodeSelectRequest := &dao.ShortCodeSelectRequest{
					Usage:  core.ShortCodeUsageRegister,
					Target: testCase.request.Email,
				}

				daoShortCodeSelect.EXPECT().
					Exec(mock.Anything, shortCodeSelectRequest).
					Return(
						testCase.daoShortCodeSelectMock.resp,
						testCase.daoShortCodeSelectMock.err,
					)
			}

			if testCase.serviceRegisterMock != nil {
				shortCodeCreateRegisterRequest := &core.ShortCodeCreateRegisterRequest{
					Email: testCase.request.Email,
					Lang:  testCase.request.Lang,
					Role:  testCase.request.Role,
				}

				serviceRegister.EXPECT().
					Exec(mock.Anything, shortCodeCreateRegisterRequest).
					Return(testCase.serviceRegisterMock.err)
			}

			service := core.NewCredentialsReconcileRole(
				daoSelect,
				daoUpdate,
				daoShortCodeSelect,
				serviceRegister,
			)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)
		})
	}
}
