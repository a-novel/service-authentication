package services_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/services"
	servicesmocks "github.com/a-novel/service-authentication/v2/internal/services/mocks"
)

func TestCredentialsUpdateRole(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type credentialsUpdateRoleMock struct {
		resp *dao.Credentials
		err  error
	}

	type credentialsSelectMock struct {
		resp *dao.Credentials
		err  error
	}

	testCases := []struct {
		name string

		request *services.CredentialsUpdateRoleRequest

		repositoryCredentialsSelectTargetMock *credentialsSelectMock
		repositoryCredentialsSelectCallerMock *credentialsSelectMock
		credentialsUpdateRoleMock             *credentialsUpdateRoleMock

		expect    *services.Credentials
		expectErr error
	}{
		{
			name: "UpgradeToLowerRole",

			request: &services.CredentialsUpdateRoleRequest{
				TargetUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				CurrentUserID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Role:          config.RoleAdmin,
			},

			repositoryCredentialsSelectTargetMock: &credentialsSelectMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "target@email.com",
					Role:      config.RoleUser,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			repositoryCredentialsSelectCallerMock: &credentialsSelectMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "target@email.com",
					Role:      config.RoleSuperAdmin,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			credentialsUpdateRoleMock: &credentialsUpdateRoleMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "target@email.com",
					Role:      config.RoleAdmin,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
			},

			expect: &services.Credentials{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Email:     "target@email.com",
				Role:      config.RoleAdmin,
				CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "UpgradeToSameRole",

			request: &services.CredentialsUpdateRoleRequest{
				TargetUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				CurrentUserID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Role:          config.RoleAdmin,
			},

			repositoryCredentialsSelectTargetMock: &credentialsSelectMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "target@email.com",
					Role:      config.RoleUser,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			repositoryCredentialsSelectCallerMock: &credentialsSelectMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "target@email.com",
					Role:      config.RoleSuperAdmin,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			credentialsUpdateRoleMock: &credentialsUpdateRoleMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "target@email.com",
					Role:      config.RoleAdmin,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
			},

			expect: &services.Credentials{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Email:     "target@email.com",
				Role:      config.RoleAdmin,
				CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "UpgradeToHigherRole",

			request: &services.CredentialsUpdateRoleRequest{
				TargetUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				CurrentUserID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Role:          config.RoleSuperAdmin,
			},

			repositoryCredentialsSelectTargetMock: &credentialsSelectMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "target@email.com",
					Role:      config.RoleUser,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			repositoryCredentialsSelectCallerMock: &credentialsSelectMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "target@email.com",
					Role:      config.RoleAdmin,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			expectErr: services.ErrCredentialsUpdateRoleToHigher,
		},
		{
			name: "DowngradeFromLowerRole",

			request: &services.CredentialsUpdateRoleRequest{
				TargetUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				CurrentUserID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Role:          config.RoleUser,
			},

			repositoryCredentialsSelectTargetMock: &credentialsSelectMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "target@email.com",
					Role:      config.RoleAdmin,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			repositoryCredentialsSelectCallerMock: &credentialsSelectMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "target@email.com",
					Role:      config.RoleSuperAdmin,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			credentialsUpdateRoleMock: &credentialsUpdateRoleMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "target@email.com",
					Role:      config.RoleUser,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
			},

			expect: &services.Credentials{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Email:     "target@email.com",
				Role:      config.RoleUser,
				CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "DowngradeFromSameRole",

			request: &services.CredentialsUpdateRoleRequest{
				TargetUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				CurrentUserID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Role:          config.RoleUser,
			},

			repositoryCredentialsSelectTargetMock: &credentialsSelectMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "target@email.com",
					Role:      config.RoleAdmin,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			repositoryCredentialsSelectCallerMock: &credentialsSelectMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "target@email.com",
					Role:      config.RoleAdmin,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			expectErr: services.ErrCredentialsUpdateRoleDowngradeSuperior,
		},

		{
			name: "UnknownRole",

			request: &services.CredentialsUpdateRoleRequest{
				TargetUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				CurrentUserID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Role:          "foo",
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "SelfUpgrade",

			request: &services.CredentialsUpdateRoleRequest{
				TargetUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				CurrentUserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Role:          config.RoleSuperAdmin,
			},

			expectErr: services.ErrCredentialsUpdateRoleSelfUpdate,
		},

		{
			name: "UpdateRoleError",

			request: &services.CredentialsUpdateRoleRequest{
				TargetUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				CurrentUserID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Role:          config.RoleAdmin,
			},

			repositoryCredentialsSelectTargetMock: &credentialsSelectMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "target@email.com",
					Role:      config.RoleUser,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			repositoryCredentialsSelectCallerMock: &credentialsSelectMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "target@email.com",
					Role:      config.RoleSuperAdmin,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			credentialsUpdateRoleMock: &credentialsUpdateRoleMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "SelectCurrentCredentialsError",

			request: &services.CredentialsUpdateRoleRequest{
				TargetUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				CurrentUserID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Role:          config.RoleAdmin,
			},

			repositoryCredentialsSelectTargetMock: &credentialsSelectMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "target@email.com",
					Role:      config.RoleUser,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			repositoryCredentialsSelectCallerMock: &credentialsSelectMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "SelectTargetCredentialsError",

			request: &services.CredentialsUpdateRoleRequest{
				TargetUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				CurrentUserID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Role:          config.RoleAdmin,
			},

			repositoryCredentialsSelectTargetMock: &credentialsSelectMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			repository := servicesmocks.NewMockCredentialsUpdateRoleRepository(t)
			repositoryCredentialsSelect := servicesmocks.NewMockCredentialsUpdateRoleRepositoryCredentialsSelect(t)

			if testCase.repositoryCredentialsSelectTargetMock != nil {
				repositoryCredentialsSelect.EXPECT().
					Exec(mock.Anything, &dao.CredentialsSelectRequest{
						ID: testCase.request.TargetUserID,
					}).
					Return(
						testCase.repositoryCredentialsSelectTargetMock.resp,
						testCase.repositoryCredentialsSelectTargetMock.err,
					)
			}

			if testCase.repositoryCredentialsSelectCallerMock != nil {
				repositoryCredentialsSelect.EXPECT().
					Exec(mock.Anything, &dao.CredentialsSelectRequest{
						ID: testCase.request.CurrentUserID,
					}).
					Return(
						testCase.repositoryCredentialsSelectCallerMock.resp,
						testCase.repositoryCredentialsSelectCallerMock.err,
					)
			}

			if testCase.credentialsUpdateRoleMock != nil {
				repository.EXPECT().
					Exec(
						mock.Anything,
						mock.MatchedBy(func(data *dao.CredentialsUpdateRoleRequest) bool {
							return assert.Equal(t, testCase.request.Role, data.Role) &&
								assert.WithinDuration(t, time.Now(), data.Now, time.Second) &&
								assert.Equal(t, testCase.request.TargetUserID, data.ID)
						}),
					).
					Return(testCase.credentialsUpdateRoleMock.resp, testCase.credentialsUpdateRoleMock.err)
			}

			service := services.NewCredentialsUpdateRole(repository, repositoryCredentialsSelect)

			resp, err := service.Exec(ctx, testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			repository.AssertExpectations(t)
			repositoryCredentialsSelect.AssertExpectations(t)
		})
	}
}
