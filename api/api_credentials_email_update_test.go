package api_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/authentication/api"
	"github.com/a-novel/authentication/api/codegen"
	apimocks "github.com/a-novel/authentication/api/mocks"
	"github.com/a-novel/authentication/internal/dao"
	"github.com/a-novel/authentication/internal/services"
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

		form *codegen.UpdateEmailForm

		updateEmailData *updateEmailData

		expect    codegen.UpdateEmailRes
		expectErr error
	}{
		{
			name: "Success",

			form: &codegen.UpdateEmailForm{
				UserID:    codegen.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				ShortCode: "foobarqux",
			},

			updateEmailData: &updateEmailData{
				resp: &services.UpdateEmailResponse{
					NewEmail: "user@provider.com",
				},
			},

			expect: &codegen.UpdateEmailOK{Email: codegen.Email("user@provider.com")},
		},
		{
			name: "UserNotFound",

			form: &codegen.UpdateEmailForm{
				UserID:    codegen.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				ShortCode: "foobarqux",
			},

			updateEmailData: &updateEmailData{
				err: dao.ErrCredentialsNotFound,
			},

			expect: &codegen.NotFoundError{Error: "user not found"},
		},
		{
			name: "ShortCodeNotFound",

			form: &codegen.UpdateEmailForm{
				UserID:    codegen.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				ShortCode: "foobarqux",
			},

			updateEmailData: &updateEmailData{
				err: dao.ErrShortCodeNotFound,
			},

			expect: &codegen.ForbiddenError{Error: "invalid short code"},
		},
		{
			name: "InvalidShortCode",

			form: &codegen.UpdateEmailForm{
				UserID:    codegen.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				ShortCode: "foobarqux",
			},

			updateEmailData: &updateEmailData{
				err: services.ErrInvalidShortCode,
			},

			expect: &codegen.ForbiddenError{Error: "invalid short code"},
		},
		{
			name: "EmailAlreadyTaken",

			form: &codegen.UpdateEmailForm{
				UserID:    codegen.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				ShortCode: "foobarqux",
			},

			updateEmailData: &updateEmailData{
				err: dao.ErrCredentialsAlreadyExists,
			},

			expect: &codegen.ConflictError{Error: "email already taken"},
		},
		{
			name: "Error",

			form: &codegen.UpdateEmailForm{
				UserID:    codegen.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
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
					UpdateEmail(t.Context(), services.UpdateEmailRequest{
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
