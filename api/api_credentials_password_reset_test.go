package api_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/authentication/api"
	"github.com/a-novel/authentication/api/codegen"
	apimocks "github.com/a-novel/authentication/api/mocks"
	"github.com/a-novel/authentication/internal/lib"
)

func TestResetPassword(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type updatePasswordData struct {
		err error
	}

	testCases := []struct {
		name string

		req *codegen.ResetPasswordForm

		updatePasswordData *updatePasswordData

		expect    codegen.ResetPasswordRes
		expectErr error
	}{
		{
			name: "Success",

			req: &codegen.ResetPasswordForm{
				Password:  "secret",
				ShortCode: "foobarqux",
				UserID:    codegen.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
			},

			updatePasswordData: &updatePasswordData{},

			expect: &codegen.ResetPasswordNoContent{},
		},
		{
			name: "ErrInvalidPassword",

			req: &codegen.ResetPasswordForm{
				Password:  "secret",
				ShortCode: "foobarqux",
				UserID:    codegen.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
			},

			updatePasswordData: &updatePasswordData{
				err: lib.ErrInvalidPassword,
			},

			expect: &codegen.ForbiddenError{Error: "invalid short code"},
		},
		{
			name: "Error",

			req: &codegen.ResetPasswordForm{
				Password:  "secret",
				ShortCode: "foobarqux",
				UserID:    codegen.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
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

			source := apimocks.NewMockUpdatePasswordService(t)

			if testCase.updatePasswordData != nil {
				source.EXPECT().
					UpdatePassword(t.Context(), mock.AnythingOfType("services.UpdatePasswordRequest")).
					Return(testCase.updatePasswordData.err)
			}

			handler := api.API{UpdatePasswordService: source}

			res, err := handler.ResetPassword(t.Context(), testCase.req)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)

			source.AssertExpectations(t)
		})
	}
}
