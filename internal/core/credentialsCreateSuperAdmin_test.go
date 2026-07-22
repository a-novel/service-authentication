package core_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"
	"github.com/a-novel-kit/golib/transaction/transactiontest"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/config/configtest"
	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/lib"
)

func TestCredentialsCreateSuperAdmin(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type daoMock struct {
		resp *dao.Credentials
		err  error
	}

	type daoSelectMock struct {
		resp *dao.Credentials
		err  error
	}

	type daoUpdatePasswordMock struct {
		resp *dao.Credentials
		err  error
	}

	type daoUpdateRoleMock struct {
		resp *dao.Credentials
		err  error
	}

	testCases := []struct {
		name string

		request *core.CredentialsCreateSuperAdminRequest

		daoMock               *daoMock
		daoSelectMock         *daoSelectMock
		daoUpdatePasswordMock *daoUpdatePasswordMock
		daoUpdateRoleMock     *daoUpdateRoleMock

		expect    *core.Credentials
		expectErr error
	}{
		{
			name: "Success",

			request: &core.CredentialsCreateSuperAdminRequest{
				Email:    "superadmin@provider.com",
				Password: "Louvre",
			},

			daoSelectMock: &daoSelectMock{
				err: dao.ErrCredentialsSelectByEmailNotFound,
			},
			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "superadmin@provider.com",
					Password:  "Louvre",
					Role:      config.RoleSuperAdmin,
					CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			expect: &core.Credentials{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Email:     "superadmin@provider.com",
				Role:      config.RoleSuperAdmin,
				CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "Success/AlreadyExists",

			request: &core.CredentialsCreateSuperAdminRequest{
				Email:    "superadmin@provider.com",
				Password: "Louvre",
			},

			daoSelectMock: &daoSelectMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "superadmin@provider.com",
					Password:  "abcdef",
					Role:      config.RoleSuperAdmin,
					CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			daoUpdatePasswordMock: &daoUpdatePasswordMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "superadmin@provider.com",
					Password:  "abcdef",
					Role:      config.RoleSuperAdmin,
					CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			expect: &core.Credentials{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Email:     "superadmin@provider.com",
				Role:      config.RoleSuperAdmin,
				CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "Success/AlreadyExists/DifferentRole",

			request: &core.CredentialsCreateSuperAdminRequest{
				Email:    "superadmin@provider.com",
				Password: "Louvre",
			},

			daoSelectMock: &daoSelectMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "superadmin@provider.com",
					Password:  "abcdef",
					Role:      config.RoleUser,
					CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			daoUpdatePasswordMock: &daoUpdatePasswordMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "superadmin@provider.com",
					Password:  "abcdef",
					Role:      config.RoleUser,
					CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			daoUpdateRoleMock: &daoUpdateRoleMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "superadmin@provider.com",
					Password:  "abcdef",
					Role:      config.RoleSuperAdmin,
					CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			expect: &core.Credentials{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Email:     "superadmin@provider.com",
				Role:      config.RoleSuperAdmin,
				CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},

		{
			name: "Error/Select",

			request: &core.CredentialsCreateSuperAdminRequest{
				Email:    "superadmin@provider.com",
				Password: "Louvre",
			},

			daoSelectMock: &daoSelectMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/Insert",

			request: &core.CredentialsCreateSuperAdminRequest{
				Email:    "superadmin@provider.com",
				Password: "Louvre",
			},

			daoSelectMock: &daoSelectMock{
				err: dao.ErrCredentialsSelectByEmailNotFound,
			},
			daoMock: &daoMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/UpdatePassword",

			request: &core.CredentialsCreateSuperAdminRequest{
				Email:    "superadmin@provider.com",
				Password: "Louvre",
			},

			daoSelectMock: &daoSelectMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "superadmin@provider.com",
					Password:  "abcdef",
					Role:      config.RoleSuperAdmin,
					CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			daoUpdatePasswordMock: &daoUpdatePasswordMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/UpdateRole",

			request: &core.CredentialsCreateSuperAdminRequest{
				Email:    "superadmin@provider.com",
				Password: "Louvre",
			},

			daoSelectMock: &daoSelectMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "superadmin@provider.com",
					Password:  "abcdef",
					Role:      config.RoleUser,
					CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			daoUpdatePasswordMock: &daoUpdatePasswordMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "superadmin@provider.com",
					Password:  "abcdef",
					Role:      config.RoleUser,
					CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			daoUpdateRoleMock: &daoUpdateRoleMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/PasswordTooShort",

			request: &core.CredentialsCreateSuperAdminRequest{
				Email:    "superadmin@provider.com",
				Password: "abc", // 3 characters, minimum is 4
			},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Success/PasswordAtMinLength",

			request: &core.CredentialsCreateSuperAdminRequest{
				Email:    "superadmin@provider.com",
				Password: "abcd", // exactly 4 characters
			},

			daoSelectMock: &daoSelectMock{
				err: dao.ErrCredentialsSelectByEmailNotFound,
			},
			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "superadmin@provider.com",
					Password:  "abcd",
					Role:      config.RoleSuperAdmin,
					CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			expect: &core.Credentials{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Email:     "superadmin@provider.com",
				Role:      config.RoleSuperAdmin,
				CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgres.RunTransactionalTest(t, configtest.PostgresPreset, func(ctx context.Context, t *testing.T) {
				t.Helper()

				mockDao := coremocks.NewMockCredentialsCreateSuperAdminDao(t)
				daoSelect := coremocks.NewMockCredentialsCreateSuperAdminDaoSelect(t)
				daoUpdatePassword := coremocks.NewMockCredentialsCreateSuperAdminDaoUpdatePassword(t)
				daoUpdateRole := coremocks.NewMockCredentialsCreateSuperAdminDaoUpdateRole(t)

				if testCase.daoMock != nil {
					mockDao.EXPECT().
						Exec(mock.Anything, mock.MatchedBy(func(data *dao.CredentialsInsertRequest) bool {
							return assert.Equal(t, testCase.request.Email, data.Email) &&
								assert.NotEqual(t, uuid.Nil, data.ID) &&
								assert.WithinDuration(t, time.Now(), data.Now, time.Minute) &&
								assert.NoError(t, lib.CompareArgon2(testCase.request.Password, data.Password)) &&
								assert.Equal(t, config.RoleSuperAdmin, data.Role)
						})).
						Return(testCase.daoMock.resp, testCase.daoMock.err)
				}

				if testCase.daoSelectMock != nil {
					daoSelect.EXPECT().
						Exec(mock.Anything, &dao.CredentialsSelectByEmailRequest{Email: testCase.request.Email}).
						Return(testCase.daoSelectMock.resp, testCase.daoSelectMock.err)
				}

				if testCase.daoUpdatePasswordMock != nil {
					daoUpdatePassword.EXPECT().
						Exec(mock.Anything, mock.MatchedBy(func(data *dao.CredentialsUpdatePasswordRequest) bool {
							return assert.Equal(t, testCase.daoSelectMock.resp.ID, data.ID) &&
								assert.WithinDuration(t, time.Now(), data.Now, time.Minute) &&
								assert.NoError(t, lib.CompareArgon2(testCase.request.Password, data.Password))
						})).
						Return(testCase.daoUpdatePasswordMock.resp, testCase.daoUpdatePasswordMock.err)
				}

				if testCase.daoUpdateRoleMock != nil {
					daoUpdateRole.EXPECT().
						Exec(mock.Anything, mock.MatchedBy(func(data *dao.CredentialsUpdateRoleRequest) bool {
							return assert.Equal(t, testCase.daoSelectMock.resp.ID, data.ID) &&
								assert.WithinDuration(t, time.Now(), data.Now, time.Minute) &&
								assert.Equal(t, config.RoleSuperAdmin, data.Role)
						})).
						Return(testCase.daoUpdateRoleMock.resp, testCase.daoUpdateRoleMock.err)
				}

				service := core.NewCredentialsCreateSuperAdmin(
					mockDao, daoSelect, daoUpdatePassword, daoUpdateRole, transactiontest.NewTransactor(),
				)

				resp, err := service.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, resp)

				mockDao.AssertExpectations(t)
				daoSelect.AssertExpectations(t)
				daoUpdatePassword.AssertExpectations(t)
				daoUpdateRole.AssertExpectations(t)
			})
		})
	}
}
