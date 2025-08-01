package api_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/internal/api"
	apimocks "github.com/a-novel/service-authentication/internal/api/mocks"
	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models/api"
)

func TestResetPassword(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type updatePasswordData struct {
		err error
	}

	testCases := []struct {
		name string

		form *apimodels.ResetPasswordForm

		updatePasswordData *updatePasswordData

		expect    apimodels.ResetPasswordRes
		expectErr error
	}{
		{
			name: "Success",

			form: &apimodels.ResetPasswordForm{
				Password:  "secret",
				ShortCode: "foobarqux",
				UserID:    apimodels.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
			},

			updatePasswordData: &updatePasswordData{},

			expect: &apimodels.ResetPasswordNoContent{},
		},
		{
			name: "ErrInvalidPassword",

			form: &apimodels.ResetPasswordForm{
				Password:  "secret",
				ShortCode: "foobarqux",
				UserID:    apimodels.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
			},

			updatePasswordData: &updatePasswordData{
				err: lib.ErrInvalidPassword,
			},

			expect: &apimodels.ForbiddenError{Error: "invalid short code"},
		},
		{
			name: "Error",

			form: &apimodels.ResetPasswordForm{
				Password:  "secret",
				ShortCode: "foobarqux",
				UserID:    apimodels.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
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
					UpdatePassword(mock.Anything, services.UpdatePasswordRequest{
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
