package api_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/internal/api"
	apimocks "github.com/a-novel/service-authentication/internal/api/mocks"
	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
	"github.com/a-novel/service-authentication/models/api"
	"github.com/a-novel/service-authentication/pkg"
)

func TestEmailUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type updatePasswordData struct {
		err error
	}

	testCases := []struct {
		name string

		userID *uuid.UUID
		form   *apimodels.UpdatePasswordForm

		updatePasswordData *updatePasswordData

		expect    apimodels.UpdatePasswordRes
		expectErr error
	}{
		{
			name: "Success",

			userID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),

			form: &apimodels.UpdatePasswordForm{
				Password:        "secret",
				CurrentPassword: "foobarqux",
			},

			updatePasswordData: &updatePasswordData{},

			expect: &apimodels.UpdatePasswordNoContent{},
		},
		{
			name: "NoUser",

			form: &apimodels.UpdatePasswordForm{
				Password:        "secret",
				CurrentPassword: "foobarqux",
			},

			expectErr: models.ErrUnauthorized,
		},
		{
			name: "ErrInvalidPassword",

			userID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),

			form: &apimodels.UpdatePasswordForm{
				Password:        "secret",
				CurrentPassword: "foobarqux",
			},

			updatePasswordData: &updatePasswordData{
				err: lib.ErrInvalidPassword,
			},

			expect: &apimodels.ForbiddenError{Error: "invalid user password"},
		},
		{
			name: "Error",

			userID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),

			form: &apimodels.UpdatePasswordForm{
				Password:        "secret",
				CurrentPassword: "foobarqux",
			},

			updatePasswordData: &updatePasswordData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.WithValue(t.Context(), pkg.ClaimsContextKey{}, &models.AccessTokenClaims{
				UserID: testCase.userID,
			})

			source := apimocks.NewMockUpdatePasswordService(t)

			if testCase.updatePasswordData != nil {
				source.EXPECT().
					UpdatePassword(mock.Anything, services.UpdatePasswordRequest{
						Password:        string(testCase.form.GetPassword()),
						CurrentPassword: string(testCase.form.GetCurrentPassword()),
						UserID:          lo.FromPtr(testCase.userID),
					}).
					Return(testCase.updatePasswordData.err)
			}

			handler := api.API{UpdatePasswordService: source}

			res, err := handler.UpdatePassword(ctx, testCase.form)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)

			source.AssertExpectations(t)
		})
	}
}
