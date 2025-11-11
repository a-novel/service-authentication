package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/golib/postgres"

	"github.com/a-novel/service-authentication/internal/config"
	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/internal/services"
	servicesmocks "github.com/a-novel/service-authentication/internal/services/mocks"
)

func TestCredentialsCreateSuperAdmin(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type repositoryMock struct {
		resp *dao.Credentials
		err  error
	}

	type repositorySelectMock struct {
		resp *dao.Credentials
		err  error
	}

	type repositoryUpdatePasswordMock struct {
		resp *dao.Credentials
		err  error
	}

	type repositoryUpdateRoleMock struct {
		resp *dao.Credentials
		err  error
	}

	testCases := []struct {
		name string

		request *services.CredentialsCreateSuperAdminRequest

		repositoryMock               *repositoryMock
		repositorySelectMock         *repositorySelectMock
		repositoryUpdatePasswordMock *repositoryUpdatePasswordMock
		repositoryUpdateRoleMock     *repositoryUpdateRoleMock

		expect    *services.Credentials
		expectErr error
	}{
		{
			name: "Success",

			request: &services.CredentialsCreateSuperAdminRequest{
				Email:    "superadmin@provider.com",
				Password: "Louvre",
			},

			repositorySelectMock: &repositorySelectMock{
				err: dao.ErrCredentialsSelectByEmailNotFound,
			},
			repositoryMock: &repositoryMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "superadmin@provider.com",
					Password:  "Louvre",
					Role:      config.RoleSuperAdmin,
					CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			expect: &services.Credentials{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Email:     "superadmin@provider.com",
				Role:      config.RoleSuperAdmin,
				CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "Success/AlreadyExists",

			request: &services.CredentialsCreateSuperAdminRequest{
				Email:    "superadmin@provider.com",
				Password: "Louvre",
			},

			repositorySelectMock: &repositorySelectMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "superadmin@provider.com",
					Password:  "abcdef",
					Role:      config.RoleSuperAdmin,
					CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			repositoryUpdatePasswordMock: &repositoryUpdatePasswordMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "superadmin@provider.com",
					Password:  "abcdef",
					Role:      config.RoleSuperAdmin,
					CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			expect: &services.Credentials{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Email:     "superadmin@provider.com",
				Role:      config.RoleSuperAdmin,
				CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "Success/AlreadyExists/DifferentRole",

			request: &services.CredentialsCreateSuperAdminRequest{
				Email:    "superadmin@provider.com",
				Password: "Louvre",
			},

			repositorySelectMock: &repositorySelectMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "superadmin@provider.com",
					Password:  "abcdef",
					Role:      config.RoleUser,
					CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			repositoryUpdatePasswordMock: &repositoryUpdatePasswordMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "superadmin@provider.com",
					Password:  "abcdef",
					Role:      config.RoleUser,
					CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			repositoryUpdateRoleMock: &repositoryUpdateRoleMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "superadmin@provider.com",
					Password:  "abcdef",
					Role:      config.RoleSuperAdmin,
					CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			expect: &services.Credentials{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Email:     "superadmin@provider.com",
				Role:      config.RoleSuperAdmin,
				CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},

		{
			name: "Error/Select",

			request: &services.CredentialsCreateSuperAdminRequest{
				Email:    "superadmin@provider.com",
				Password: "Louvre",
			},

			repositorySelectMock: &repositorySelectMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/Insert",

			request: &services.CredentialsCreateSuperAdminRequest{
				Email:    "superadmin@provider.com",
				Password: "Louvre",
			},

			repositorySelectMock: &repositorySelectMock{
				err: dao.ErrCredentialsSelectByEmailNotFound,
			},
			repositoryMock: &repositoryMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/UpdatePassword",

			request: &services.CredentialsCreateSuperAdminRequest{
				Email:    "superadmin@provider.com",
				Password: "Louvre",
			},

			repositorySelectMock: &repositorySelectMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "superadmin@provider.com",
					Password:  "abcdef",
					Role:      config.RoleSuperAdmin,
					CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			repositoryUpdatePasswordMock: &repositoryUpdatePasswordMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/UpdateRole",

			request: &services.CredentialsCreateSuperAdminRequest{
				Email:    "superadmin@provider.com",
				Password: "Louvre",
			},

			repositorySelectMock: &repositorySelectMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "superadmin@provider.com",
					Password:  "abcdef",
					Role:      config.RoleUser,
					CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			repositoryUpdatePasswordMock: &repositoryUpdatePasswordMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "superadmin@provider.com",
					Password:  "abcdef",
					Role:      config.RoleUser,
					CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			repositoryUpdateRoleMock: &repositoryUpdateRoleMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgres.RunTransactionalTest(t, config.PostgresPresetTest, func(ctx context.Context, t *testing.T) {
				t.Helper()

				repository := servicesmocks.NewMockCredentialsCreateSuperAdminRepository(t)
				repositorySelect := servicesmocks.NewMockCredentialsCreateSuperAdminRepositorySelect(t)
				repositoryUpdatePassword := servicesmocks.NewMockCredentialsCreateSuperAdminRepositoryUpdatePassword(t)
				repositoryUpdateRole := servicesmocks.NewMockCredentialsCreateSuperAdminRepositoryUpdateRole(t)

				if testCase.repositoryMock != nil {
					repository.EXPECT().
						Exec(mock.Anything, mock.MatchedBy(func(data *dao.CredentialsInsertRequest) bool {
							return assert.Equal(t, testCase.request.Email, data.Email) &&
								assert.NotEqual(t, uuid.Nil, data.ID) &&
								assert.WithinDuration(t, time.Now(), data.Now, time.Second) &&
								assert.NoError(t, lib.CompareScrypt(testCase.request.Password, data.Password)) &&
								assert.Equal(t, config.RoleSuperAdmin, data.Role)
						})).
						Return(testCase.repositoryMock.resp, testCase.repositoryMock.err)
				}

				if testCase.repositorySelectMock != nil {
					repositorySelect.EXPECT().
						Exec(mock.Anything, &dao.CredentialsSelectByEmailRequest{Email: testCase.request.Email}).
						Return(testCase.repositorySelectMock.resp, testCase.repositorySelectMock.err)
				}

				if testCase.repositoryUpdatePasswordMock != nil {
					repositoryUpdatePassword.EXPECT().
						Exec(mock.Anything, mock.MatchedBy(func(data *dao.CredentialsUpdatePasswordRequest) bool {
							return assert.Equal(t, testCase.repositorySelectMock.resp.ID, data.ID) &&
								assert.WithinDuration(t, time.Now(), data.Now, time.Second) &&
								assert.NoError(t, lib.CompareScrypt(testCase.request.Password, data.Password))
						})).
						Return(testCase.repositoryUpdatePasswordMock.resp, testCase.repositoryUpdatePasswordMock.err)
				}

				if testCase.repositoryUpdateRoleMock != nil {
					repositoryUpdateRole.EXPECT().
						Exec(mock.Anything, mock.MatchedBy(func(data *dao.CredentialsUpdateRoleRequest) bool {
							return assert.Equal(t, testCase.repositorySelectMock.resp.ID, data.ID) &&
								assert.WithinDuration(t, time.Now(), data.Now, time.Second) &&
								assert.Equal(t, config.RoleSuperAdmin, data.Role)
						})).
						Return(testCase.repositoryUpdateRoleMock.resp, testCase.repositoryUpdateRoleMock.err)
				}

				service := services.NewCredentialsCreateSuperAdmin(
					repository, repositorySelect, repositoryUpdatePassword, repositoryUpdateRole,
				)

				resp, err := service.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, resp)

				repository.AssertExpectations(t)
				repositorySelect.AssertExpectations(t)
				repositoryUpdatePassword.AssertExpectations(t)
				repositoryUpdateRole.AssertExpectations(t)
			})
		})
	}
}
