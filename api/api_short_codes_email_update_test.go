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
	"github.com/a-novel/authentication/models"
)

func TestRequestEmailUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type requestEmailUpdateData struct {
		err error
	}

	testCases := []struct {
		name string

		userID *uuid.UUID
		req    *codegen.RequestEmailUpdateForm

		requestEmailUpdateData *requestEmailUpdateData

		expect    codegen.RequestEmailUpdateRes
		expectErr error
	}{
		{
			name: "Success",

			userID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),

			req: &codegen.RequestEmailUpdateForm{
				Email: "user@provider.com",
			},

			requestEmailUpdateData: &requestEmailUpdateData{},

			expect: &codegen.RequestEmailUpdateNoContent{},
		},
		{
			name: "NoUser",

			req: &codegen.RequestEmailUpdateForm{
				Email: "user@provider.com",
			},

			expectErr: api.ErrMissingUserID,
		},
		{
			name: "RequestEmailUpdateError",

			userID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),

			req: &codegen.RequestEmailUpdateForm{
				Email: "user@provider.com",
			},

			requestEmailUpdateData: &requestEmailUpdateData{
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

			source := apimocks.NewMockRequestEmailUpdateService(t)

			if testCase.requestEmailUpdateData != nil {
				source.EXPECT().
					RequestEmailUpdate(ctx, mock.AnythingOfType("services.RequestEmailUpdateRequest")).
					Return(nil, testCase.requestEmailUpdateData.err)
			}

			handler := api.API{RequestEmailUpdateService: source}

			res, err := handler.RequestEmailUpdate(ctx, testCase.req)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)

			source.AssertExpectations(t)
		})
	}
}
