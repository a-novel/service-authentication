package api_test

import (
	"context"
	"errors"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/api"
	"github.com/a-novel/service-authentication/api/codegen"
	apimocks "github.com/a-novel/service-authentication/api/mocks"
	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
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

		form *codegen.UpdateRoleForm

		updateRoleData *updateRoleData

		expect    codegen.UpdateRoleRes
		expectErr error
	}{
		{
			name: "Success",

			form: &codegen.UpdateRoleForm{
				UserID: codegen.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
				Role:   codegen.CredentialsRoleUser,
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

			expect: &codegen.User{
				ID:        codegen.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
				Email:     codegen.Email("target@mail.com"),
				Role:      codegen.CredentialsRoleUser,
				CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "CredentialsNotFound",

			form: &codegen.UpdateRoleForm{
				UserID: codegen.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				Role:   codegen.CredentialsRoleUser,
			},

			updateRoleData: &updateRoleData{
				err: dao.ErrCredentialsNotFound,
			},

			expect: &codegen.NotFoundError{Error: dao.ErrCredentialsNotFound.Error()},
		},
		{
			name: "UpdateToHigherRole",

			form: &codegen.UpdateRoleForm{
				UserID: codegen.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				Role:   codegen.CredentialsRoleUser,
			},

			updateRoleData: &updateRoleData{
				err: services.ErrUpdateToHigherRole,
			},

			expect: &codegen.ForbiddenError{Error: services.ErrUpdateToHigherRole.Error()},
		},
		{
			name: "MustDowngradeLowerRole",

			form: &codegen.UpdateRoleForm{
				UserID: codegen.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				Role:   codegen.CredentialsRoleUser,
			},

			updateRoleData: &updateRoleData{
				err: services.ErrMustDowngradeLowerRole,
			},

			expect: &codegen.ForbiddenError{Error: services.ErrMustDowngradeLowerRole.Error()},
		},
		{
			name: "UnknownRole",

			form: &codegen.UpdateRoleForm{
				UserID: codegen.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				Role:   codegen.CredentialsRoleUser,
			},

			updateRoleData: &updateRoleData{
				err: services.ErrUnknownRole,
			},

			expect: &codegen.UnprocessableEntityError{Error: services.ErrUnknownRole.Error()},
		},
		{
			name: "SelfRoleUpdate",

			form: &codegen.UpdateRoleForm{
				UserID: codegen.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				Role:   codegen.CredentialsRoleUser,
			},

			updateRoleData: &updateRoleData{
				err: services.ErrSelfRoleUpdate,
			},

			expect: &codegen.UnprocessableEntityError{Error: services.ErrSelfRoleUpdate.Error()},
		},
		{
			name: "Error",

			form: &codegen.UpdateRoleForm{
				UserID: codegen.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				Role:   codegen.CredentialsRoleUser,
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

			ctx := context.WithValue(t.Context(), api.ClaimsAPIKey{}, &models.AccessTokenClaims{
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
