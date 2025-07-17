package api_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/internal/api"
	apimocks "github.com/a-novel/service-authentication/internal/api/mocks"
	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
	"github.com/a-novel/service-authentication/models/api"
	"github.com/a-novel/service-authentication/pkg"
)

func TestUpdateRole(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type updateRoleData struct {
		resp *models.User
		err  error
	}

	testCases := []struct {
		name string

		form *apimodels.UpdateRoleForm

		updateRoleData *updateRoleData

		expect    apimodels.UpdateRoleRes
		expectErr error
	}{
		{
			name: "Success",

			form: &apimodels.UpdateRoleForm{
				UserID: apimodels.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
				Role:   apimodels.CredentialsRoleUser,
			},

			updateRoleData: &updateRoleData{
				resp: &models.User{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "target@mail.com",
					Role:      models.CredentialsRoleUser,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			expect: &apimodels.User{
				ID:        apimodels.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
				Email:     apimodels.Email("target@mail.com"),
				Role:      apimodels.CredentialsRoleUser,
				CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "CredentialsNotFound",

			form: &apimodels.UpdateRoleForm{
				UserID: apimodels.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				Role:   apimodels.CredentialsRoleUser,
			},

			updateRoleData: &updateRoleData{
				err: dao.ErrCredentialsNotFound,
			},

			expect: &apimodels.NotFoundError{Error: dao.ErrCredentialsNotFound.Error()},
		},
		{
			name: "UpdateToHigherRole",

			form: &apimodels.UpdateRoleForm{
				UserID: apimodels.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				Role:   apimodels.CredentialsRoleUser,
			},

			updateRoleData: &updateRoleData{
				err: services.ErrUpdateToHigherRole,
			},

			expect: &apimodels.ForbiddenError{Error: services.ErrUpdateToHigherRole.Error()},
		},
		{
			name: "MustDowngradeLowerRole",

			form: &apimodels.UpdateRoleForm{
				UserID: apimodels.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				Role:   apimodels.CredentialsRoleUser,
			},

			updateRoleData: &updateRoleData{
				err: services.ErrMustDowngradeLowerRole,
			},

			expect: &apimodels.ForbiddenError{Error: services.ErrMustDowngradeLowerRole.Error()},
		},
		{
			name: "UnknownRole",

			form: &apimodels.UpdateRoleForm{
				UserID: apimodels.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				Role:   apimodels.CredentialsRoleUser,
			},

			updateRoleData: &updateRoleData{
				err: services.ErrUnknownRole,
			},

			expect: &apimodels.UnprocessableEntityError{Error: services.ErrUnknownRole.Error()},
		},
		{
			name: "SelfRoleUpdate",

			form: &apimodels.UpdateRoleForm{
				UserID: apimodels.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				Role:   apimodels.CredentialsRoleUser,
			},

			updateRoleData: &updateRoleData{
				err: services.ErrSelfRoleUpdate,
			},

			expect: &apimodels.UnprocessableEntityError{Error: services.ErrSelfRoleUpdate.Error()},
		},
		{
			name: "Error",

			form: &apimodels.UpdateRoleForm{
				UserID: apimodels.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				Role:   apimodels.CredentialsRoleUser,
			},

			updateRoleData: &updateRoleData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			source := apimocks.NewMockUpdateRoleService(t)

			ctx := context.WithValue(t.Context(), pkg.ClaimsContextKey{}, &models.AccessTokenClaims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-1000-0000-0000-000000000001")),
			})

			if testCase.updateRoleData != nil {
				source.EXPECT().
					UpdateRole(mock.Anything, services.UpdateRoleRequest{
						TargetUserID:  uuid.UUID(testCase.form.GetUserID()),
						CurrentUserID: uuid.MustParse("00000000-1000-0000-0000-000000000001"),
						Role:          new(api.API).CredentialsRoleToModel(testCase.form.GetRole()),
					}).
					Return(testCase.updateRoleData.resp, testCase.updateRoleData.err)
			}

			handler := api.API{UpdateRoleService: source}

			res, err := handler.UpdateRole(ctx, testCase.form)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)

			source.AssertExpectations(t)
		})
	}
}
