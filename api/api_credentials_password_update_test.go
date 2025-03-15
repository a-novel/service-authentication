package api_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/context"

	"github.com/a-novel/authentication/api"
	"github.com/a-novel/authentication/api/codegen"
	apimocks "github.com/a-novel/authentication/api/mocks"
	"github.com/a-novel/authentication/internal/lib"
	"github.com/a-novel/authentication/models"
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
		req    *codegen.UpdatePasswordForm

		updatePasswordData *updatePasswordData

		expect    codegen.UpdatePasswordRes
		expectErr error
	}{
		{
			name: "Success",

			userID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),

			req: &codegen.UpdatePasswordForm{
				Password:        "secret",
				CurrentPassword: "foobarqux",
			},

			updatePasswordData: &updatePasswordData{},

			expect: &codegen.UpdatePasswordNoContent{},
		},
		{
			name: "NoUser",

			req: &codegen.UpdatePasswordForm{
				Password:        "secret",
				CurrentPassword: "foobarqux",
			},

			expectErr: api.ErrMissingUserID,
		},
		{
			name: "ErrInvalidPassword",

			userID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),

			req: &codegen.UpdatePasswordForm{
				Password:        "secret",
				CurrentPassword: "foobarqux",
			},

			updatePasswordData: &updatePasswordData{
				err: lib.ErrInvalidPassword,
			},

			expect: &codegen.ForbiddenError{Error: "invalid user password"},
		},
		{
			name: "Error",

			userID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),

			req: &codegen.UpdatePasswordForm{
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

			ctx := context.WithValue(t.Context(), api.ClaimsAPIKey{}, &models.AccessTokenClaims{
				UserID: testCase.userID,
			})

			source := apimocks.NewMockUpdatePasswordService(t)

			if testCase.updatePasswordData != nil {
				source.EXPECT().
					UpdatePassword(ctx, mock.AnythingOfType("services.UpdatePasswordRequest")).
					Return(testCase.updatePasswordData.err)
			}

			handler := api.API{UpdatePasswordService: source}

			res, err := handler.UpdatePassword(ctx, testCase.req)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)

			source.AssertExpectations(t)
		})
	}
}
