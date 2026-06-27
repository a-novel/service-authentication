package core_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
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

		request *core.CredentialsUpdateRoleRequest

		daoCredentialsSelectTargetMock *credentialsSelectMock
		daoCredentialsSelectCallerMock *credentialsSelectMock
		credentialsUpdateRoleMock      *credentialsUpdateRoleMock

		expect    *core.Credentials
		expectErr error
	}{
		{
			name: "UpgradeToLowerRole",

			request: &core.CredentialsUpdateRoleRequest{
				TargetUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				CurrentUserID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Role:          config.RoleAdmin,
			},

			daoCredentialsSelectTargetMock: &credentialsSelectMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "target@email.com",
					Role:      config.RoleUser,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			daoCredentialsSelectCallerMock: &credentialsSelectMock{
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

			expect: &core.Credentials{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Email:     "target@email.com",
				Role:      config.RoleAdmin,
				CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "UpgradeToSameRole",

			request: &core.CredentialsUpdateRoleRequest{
				TargetUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				CurrentUserID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Role:          config.RoleAdmin,
			},

			daoCredentialsSelectTargetMock: &credentialsSelectMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "target@email.com",
					Role:      config.RoleUser,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			daoCredentialsSelectCallerMock: &credentialsSelectMock{
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

			expect: &core.Credentials{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Email:     "target@email.com",
				Role:      config.RoleAdmin,
				CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "UpgradeToHigherRole",

			request: &core.CredentialsUpdateRoleRequest{
				TargetUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				CurrentUserID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Role:          config.RoleSuperAdmin,
			},

			daoCredentialsSelectTargetMock: &credentialsSelectMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "target@email.com",
					Role:      config.RoleUser,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			daoCredentialsSelectCallerMock: &credentialsSelectMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "target@email.com",
					Role:      config.RoleAdmin,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			expectErr: core.ErrCredentialsUpdateRoleToHigher,
		},
		{
			name: "DowngradeFromLowerRole",

			request: &core.CredentialsUpdateRoleRequest{
				TargetUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				CurrentUserID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Role:          config.RoleUser,
			},

			daoCredentialsSelectTargetMock: &credentialsSelectMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "target@email.com",
					Role:      config.RoleAdmin,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			daoCredentialsSelectCallerMock: &credentialsSelectMock{
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

			expect: &core.Credentials{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Email:     "target@email.com",
				Role:      config.RoleUser,
				CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "DowngradeFromSameRole",

			request: &core.CredentialsUpdateRoleRequest{
				TargetUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				CurrentUserID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Role:          config.RoleUser,
			},

			daoCredentialsSelectTargetMock: &credentialsSelectMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "target@email.com",
					Role:      config.RoleAdmin,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			daoCredentialsSelectCallerMock: &credentialsSelectMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "target@email.com",
					Role:      config.RoleAdmin,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			expectErr: core.ErrCredentialsUpdateRoleDowngradeSuperior,
		},

		{
			name: "UnknownRole",

			request: &core.CredentialsUpdateRoleRequest{
				TargetUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				CurrentUserID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Role:          "foo",
			},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "SelfUpgrade",

			request: &core.CredentialsUpdateRoleRequest{
				TargetUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				CurrentUserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Role:          config.RoleSuperAdmin,
			},

			expectErr: core.ErrCredentialsUpdateRoleSelfUpdate,
		},

		{
			name: "UpdateRoleError",

			request: &core.CredentialsUpdateRoleRequest{
				TargetUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				CurrentUserID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Role:          config.RoleAdmin,
			},

			daoCredentialsSelectTargetMock: &credentialsSelectMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "target@email.com",
					Role:      config.RoleUser,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			daoCredentialsSelectCallerMock: &credentialsSelectMock{
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

			request: &core.CredentialsUpdateRoleRequest{
				TargetUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				CurrentUserID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Role:          config.RoleAdmin,
			},

			daoCredentialsSelectTargetMock: &credentialsSelectMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "target@email.com",
					Role:      config.RoleUser,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			daoCredentialsSelectCallerMock: &credentialsSelectMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "SelectTargetCredentialsError",

			request: &core.CredentialsUpdateRoleRequest{
				TargetUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				CurrentUserID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Role:          config.RoleAdmin,
			},

			daoCredentialsSelectTargetMock: &credentialsSelectMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			mockDao := coremocks.NewMockCredentialsUpdateRoleDao(t)
			daoCredentialsSelect := coremocks.NewMockCredentialsUpdateRoleDaoCredentialsSelect(t)

			if testCase.daoCredentialsSelectTargetMock != nil {
				daoCredentialsSelect.EXPECT().
					Exec(mock.Anything, &dao.CredentialsSelectRequest{
						ID: testCase.request.TargetUserID,
					}).
					Return(
						testCase.daoCredentialsSelectTargetMock.resp,
						testCase.daoCredentialsSelectTargetMock.err,
					)
			}

			if testCase.daoCredentialsSelectCallerMock != nil {
				daoCredentialsSelect.EXPECT().
					Exec(mock.Anything, &dao.CredentialsSelectRequest{
						ID: testCase.request.CurrentUserID,
					}).
					Return(
						testCase.daoCredentialsSelectCallerMock.resp,
						testCase.daoCredentialsSelectCallerMock.err,
					)
			}

			if testCase.credentialsUpdateRoleMock != nil {
				mockDao.EXPECT().
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

			service := core.NewCredentialsUpdateRole(mockDao, daoCredentialsSelect)

			resp, err := service.Exec(ctx, testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			mockDao.AssertExpectations(t)
			daoCredentialsSelect.AssertExpectations(t)
		})
	}
}
