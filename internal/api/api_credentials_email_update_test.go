package api_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/internal/api"
	apimocks "github.com/a-novel/service-authentication/internal/api/mocks"
	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models/api"
)

func TestUpdateEmail(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type updateEmailData struct {
		resp *services.UpdateEmailResponse
		err  error
	}

	testCases := []struct {
		name string

		form *apimodels.UpdateEmailForm

		updateEmailData *updateEmailData

		expect    apimodels.UpdateEmailRes
		expectErr error
	}{
		{
			name: "Success",

			form: &apimodels.UpdateEmailForm{
				UserID:    apimodels.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				ShortCode: "foobarqux",
			},

			updateEmailData: &updateEmailData{
				resp: &services.UpdateEmailResponse{
					NewEmail: "user@provider.com",
				},
			},

			expect: &apimodels.NewEmail{Email: "user@provider.com"},
		},
		{
			name: "UserNotFound",

			form: &apimodels.UpdateEmailForm{
				UserID:    apimodels.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				ShortCode: "foobarqux",
			},

			updateEmailData: &updateEmailData{
				err: dao.ErrCredentialsNotFound,
			},

			expect: &apimodels.NotFoundError{Error: "user not found"},
		},
		{
			name: "ShortCodeNotFound",

			form: &apimodels.UpdateEmailForm{
				UserID:    apimodels.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				ShortCode: "foobarqux",
			},

			updateEmailData: &updateEmailData{
				err: dao.ErrShortCodeNotFound,
			},

			expect: &apimodels.ForbiddenError{Error: "invalid short code"},
		},
		{
			name: "InvalidShortCode",

			form: &apimodels.UpdateEmailForm{
				UserID:    apimodels.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				ShortCode: "foobarqux",
			},

			updateEmailData: &updateEmailData{
				err: services.ErrInvalidShortCode,
			},

			expect: &apimodels.ForbiddenError{Error: "invalid short code"},
		},
		{
			name: "EmailAlreadyTaken",

			form: &apimodels.UpdateEmailForm{
				UserID:    apimodels.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				ShortCode: "foobarqux",
			},

			updateEmailData: &updateEmailData{
				err: dao.ErrCredentialsAlreadyExists,
			},

			expect: &apimodels.ConflictError{Error: "email already taken"},
		},
		{
			name: "Error",

			form: &apimodels.UpdateEmailForm{
				UserID:    apimodels.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				ShortCode: "foobarqux",
			},

			updateEmailData: &updateEmailData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			source := apimocks.NewMockUpdateEmailService(t)

			if testCase.updateEmailData != nil {
				source.EXPECT().
					UpdateEmail(mock.Anything, services.UpdateEmailRequest{
						UserID:    uuid.UUID(testCase.form.GetUserID()),
						ShortCode: string(testCase.form.GetShortCode()),
					}).
					Return(testCase.updateEmailData.resp, testCase.updateEmailData.err)
			}

			handler := api.API{UpdateEmailService: source}

			res, err := handler.UpdateEmail(t.Context(), testCase.form)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)

			source.AssertExpectations(t)
		})
	}
}
