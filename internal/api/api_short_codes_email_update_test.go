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
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
	"github.com/a-novel/service-authentication/models/api"
	"github.com/a-novel/service-authentication/pkg"
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
		form   *apimodels.RequestEmailUpdateForm

		requestEmailUpdateData *requestEmailUpdateData

		expect    apimodels.RequestEmailUpdateRes
		expectErr error
	}{
		{
			name: "Success",

			userID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),

			form: &apimodels.RequestEmailUpdateForm{
				Email: "user@provider.com",
			},

			requestEmailUpdateData: &requestEmailUpdateData{},

			expect: &apimodels.RequestEmailUpdateNoContent{},
		},
		{
			name: "Success/Lang",

			userID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),

			form: &apimodels.RequestEmailUpdateForm{
				Email: "user@provider.com",
				Lang:  apimodels.OptLang{Value: apimodels.LangFr, Set: true},
			},

			requestEmailUpdateData: &requestEmailUpdateData{},

			expect: &apimodels.RequestEmailUpdateNoContent{},
		},
		{
			name: "NoUser",

			form: &apimodels.RequestEmailUpdateForm{
				Email: "user@provider.com",
			},

			expectErr: models.ErrUnauthorized,
		},
		{
			name: "RequestEmailUpdateError",

			userID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),

			form: &apimodels.RequestEmailUpdateForm{
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

			ctx := context.WithValue(t.Context(), pkg.ClaimsContextKey{}, &models.AccessTokenClaims{
				UserID: testCase.userID,
			})

			source := apimocks.NewMockRequestEmailUpdateService(t)

			if testCase.requestEmailUpdateData != nil {
				source.EXPECT().
					RequestEmailUpdate(mock.Anything, services.RequestEmailUpdateRequest{
						Email: string(testCase.form.GetEmail()),
						Lang:  models.Lang(testCase.form.GetLang().Value),
						ID:    lo.FromPtr(testCase.userID),
					}).
					Return(nil, testCase.requestEmailUpdateData.err)
			}

			handler := api.API{RequestEmailUpdateService: source}

			res, err := handler.RequestEmailUpdate(ctx, testCase.form)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)

			source.AssertExpectations(t)
		})
	}
}
