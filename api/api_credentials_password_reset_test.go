package api_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/authentication/api"
	"github.com/a-novel/authentication/api/codegen"
	apimocks "github.com/a-novel/authentication/api/mocks"
	"github.com/a-novel/authentication/internal/lib"
	"github.com/a-novel/authentication/internal/services"
)

func TestResetPassword(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type updatePasswordData struct {
		err error
	}

	testCases := []struct {
		name string

		form *codegen.ResetPasswordForm

		updatePasswordData *updatePasswordData

		expect    codegen.ResetPasswordRes
		expectErr error
	}{
		{
			name: "Success",

			form: &codegen.ResetPasswordForm{
				Password:  "secret",
				ShortCode: "foobarqux",
				UserID:    codegen.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
			},

			updatePasswordData: &updatePasswordData{},

			expect: &codegen.ResetPasswordNoContent{},
		},
		{
			name: "ErrInvalidPassword",

			form: &codegen.ResetPasswordForm{
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

			form: &codegen.ResetPasswordForm{
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
					UpdatePassword(t.Context(), services.UpdatePasswordRequest{
						Password:  string(testCase.form.GetPassword()),
						ShortCode: string(testCase.form.GetShortCode()),
						UserID:    uuid.UUID(testCase.form.GetUserID()),
					}).
					Return(testCase.updatePasswordData.err)
			}

			handler := api.API{UpdatePasswordService: source}

			res, err := handler.ResetPassword(t.Context(), testCase.form)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)

			source.AssertExpectations(t)
		})
	}
}
