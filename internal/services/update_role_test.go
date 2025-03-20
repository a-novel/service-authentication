package services_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/authentication/internal/dao"
	"github.com/a-novel/authentication/internal/services"
	servicesmocks "github.com/a-novel/authentication/internal/services/mocks"
	"github.com/a-novel/authentication/models"
)

func TestUpdateRole(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type updateRoleData struct {
		resp *dao.CredentialsEntity
		err  error
	}

	type selectCredentialsData struct {
		resp *dao.CredentialsEntity
		err  error
	}

	testCases := []struct {
		name string

		request services.UpdateRoleRequest

		selectTargetCredentialsData  *selectCredentialsData
		selectCurrentCredentialsData *selectCredentialsData
		updateRoleData               *updateRoleData

		expect    *models.User
		expectErr error
	}{
		{
			name: "UpgradeToLowerRole",

			request: services.UpdateRoleRequest{
				TargetUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				CurrentUserID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Role:          models.CredentialsRoleAdmin,
			},

			selectTargetCredentialsData: &selectCredentialsData{
				resp: &dao.CredentialsEntity{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "target@email.com",
					Role:      models.CredentialsRoleUser,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			selectCurrentCredentialsData: &selectCredentialsData{
				resp: &dao.CredentialsEntity{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "target@email.com",
					Role:      models.CredentialsRoleSuperAdmin,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			updateRoleData: &updateRoleData{
				resp: &dao.CredentialsEntity{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "target@email.com",
					Role:      models.CredentialsRoleAdmin,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
			},

			expect: &models.User{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Email:     "target@email.com",
				Role:      models.CredentialsRoleAdmin,
				CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "UpgradeToSameRole",

			request: services.UpdateRoleRequest{
				TargetUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				CurrentUserID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Role:          models.CredentialsRoleAdmin,
			},

			selectTargetCredentialsData: &selectCredentialsData{
				resp: &dao.CredentialsEntity{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "target@email.com",
					Role:      models.CredentialsRoleUser,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			selectCurrentCredentialsData: &selectCredentialsData{
				resp: &dao.CredentialsEntity{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "target@email.com",
					Role:      models.CredentialsRoleSuperAdmin,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			updateRoleData: &updateRoleData{
				resp: &dao.CredentialsEntity{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "target@email.com",
					Role:      models.CredentialsRoleAdmin,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
			},

			expect: &models.User{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Email:     "target@email.com",
				Role:      models.CredentialsRoleAdmin,
				CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "UpgradeToHigherRole",

			request: services.UpdateRoleRequest{
				TargetUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				CurrentUserID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Role:          models.CredentialsRoleSuperAdmin,
			},

			selectTargetCredentialsData: &selectCredentialsData{
				resp: &dao.CredentialsEntity{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "target@email.com",
					Role:      models.CredentialsRoleUser,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			selectCurrentCredentialsData: &selectCredentialsData{
				resp: &dao.CredentialsEntity{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "target@email.com",
					Role:      models.CredentialsRoleAdmin,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			expectErr: services.ErrUpdateToHigherRole,
		},
		{
			name: "DowngradeFromLowerRole",

			request: services.UpdateRoleRequest{
				TargetUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				CurrentUserID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Role:          models.CredentialsRoleUser,
			},

			selectTargetCredentialsData: &selectCredentialsData{
				resp: &dao.CredentialsEntity{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "target@email.com",
					Role:      models.CredentialsRoleAdmin,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			selectCurrentCredentialsData: &selectCredentialsData{
				resp: &dao.CredentialsEntity{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "target@email.com",
					Role:      models.CredentialsRoleSuperAdmin,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			updateRoleData: &updateRoleData{
				resp: &dao.CredentialsEntity{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "target@email.com",
					Role:      models.CredentialsRoleUser,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
			},

			expect: &models.User{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Email:     "target@email.com",
				Role:      models.CredentialsRoleUser,
				CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "DowngradeFromSameRole",

			request: services.UpdateRoleRequest{
				TargetUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				CurrentUserID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Role:          models.CredentialsRoleUser,
			},

			selectTargetCredentialsData: &selectCredentialsData{
				resp: &dao.CredentialsEntity{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "target@email.com",
					Role:      models.CredentialsRoleAdmin,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			selectCurrentCredentialsData: &selectCredentialsData{
				resp: &dao.CredentialsEntity{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "target@email.com",
					Role:      models.CredentialsRoleAdmin,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			expectErr: services.ErrMustDowngradeLowerRole,
		},

		{
			name: "UnknownRole",

			request: services.UpdateRoleRequest{
				TargetUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				CurrentUserID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Role:          "foo",
			},

			expectErr: services.ErrUnknownRole,
		},
		{
			name: "SelfUpgrade",

			request: services.UpdateRoleRequest{
				TargetUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				CurrentUserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Role:          models.CredentialsRoleSuperAdmin,
			},

			expectErr: services.ErrSelfRoleUpdate,
		},

		{
			name: "UpdateRoleError",

			request: services.UpdateRoleRequest{
				TargetUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				CurrentUserID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Role:          models.CredentialsRoleAdmin,
			},

			selectTargetCredentialsData: &selectCredentialsData{
				resp: &dao.CredentialsEntity{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "target@email.com",
					Role:      models.CredentialsRoleUser,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			selectCurrentCredentialsData: &selectCredentialsData{
				resp: &dao.CredentialsEntity{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "target@email.com",
					Role:      models.CredentialsRoleSuperAdmin,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			updateRoleData: &updateRoleData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "SelectCurrentCredentialsError",

			request: services.UpdateRoleRequest{
				TargetUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				CurrentUserID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Role:          models.CredentialsRoleAdmin,
			},

			selectTargetCredentialsData: &selectCredentialsData{
				resp: &dao.CredentialsEntity{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "target@email.com",
					Role:      models.CredentialsRoleUser,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			selectCurrentCredentialsData: &selectCredentialsData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "SelectTargetCredentialsError",

			request: services.UpdateRoleRequest{
				TargetUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				CurrentUserID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Role:          models.CredentialsRoleAdmin,
			},

			selectTargetCredentialsData: &selectCredentialsData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			source := servicesmocks.NewMockUpdateRoleSource(t)

			if testCase.selectTargetCredentialsData != nil {
				source.EXPECT().
					SelectCredentials(ctx, testCase.request.TargetUserID).
					Return(testCase.selectTargetCredentialsData.resp, testCase.selectTargetCredentialsData.err)
			}

			if testCase.selectCurrentCredentialsData != nil {
				source.EXPECT().
					SelectCredentials(ctx, testCase.request.CurrentUserID).
					Return(testCase.selectCurrentCredentialsData.resp, testCase.selectCurrentCredentialsData.err)
			}

			if testCase.updateRoleData != nil {
				source.EXPECT().
					UpdateCredentialsRole(
						ctx,
						testCase.request.TargetUserID,
						mock.MatchedBy(func(data dao.UpdateCredentialsRoleData) bool {
							return assert.Equal(t, testCase.request.Role, data.Role) &&
								assert.WithinDuration(t, time.Now(), data.Now, time.Second)
						}),
					).
					Return(testCase.updateRoleData.resp, testCase.updateRoleData.err)
			}

			service := services.NewUpdateRoleService(source)

			resp, err := service.UpdateRole(ctx, testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			source.AssertExpectations(t)
		})
	}
}
