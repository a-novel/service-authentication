package handlers_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/handlers"
	"github.com/a-novel/service-authentication/v2/internal/handlers/middlewares"
	handlersmocks "github.com/a-novel/service-authentication/v2/internal/handlers/mocks"
	"github.com/a-novel/service-authentication/v2/internal/services"
)

func TestShortCodeCreateEmailUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceMock struct {
		req  *services.ShortCodeCreateEmailUpdateRequest
		resp *services.ShortCode
		err  error
	}

	testCases := []struct {
		name string

		request *http.Request
		claims  *services.AccessTokenClaims

		serviceMock *serviceMock

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{
				"email": "new_user@provider.com",
				"lang": "fr"
			}`)),
			claims: &services.AccessTokenClaims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
			},

			serviceMock: &serviceMock{
				req: &services.ShortCodeCreateEmailUpdateRequest{
					Email: "new_user@provider.com",
					Lang:  "fr",
					ID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
				resp: &services.ShortCode{
					ID:        uuid.MustParse("00000000-0000-0000-0000-111111111111"),
					Usage:     services.ShortCodeUsageValidateEmail,
					Target:    "00000000-0000-0000-0000-000000000001",
					PlainCode: "abcdef",
				},
			},

			expectStatus: http.StatusNoContent,
		},
		{
			name: "Error/Internal",

			request: httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{
				"email": "new_user@provider.com",
				"lang": "fr"
			}`)),
			claims: &services.AccessTokenClaims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
			},

			serviceMock: &serviceMock{
				req: &services.ShortCodeCreateEmailUpdateRequest{
					Email: "new_user@provider.com",
					Lang:  "fr",
					ID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
				err: errFoo,
			},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockShortCodeCreateEmailUpdateService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, testCase.serviceMock.req).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewShortCodeCreateEmailUpdate(service)
			w := httptest.NewRecorder()

			rCtx := testCase.request.Context()
			rCtx = middlewares.SetClaimsContext(rCtx, testCase.claims)

			handler.ServeHTTP(w, testCase.request.WithContext(rCtx))

			res := w.Result()

			require.Equal(t, testCase.expectStatus, res.StatusCode)

			if testCase.expectResponse != nil {
				data, err := io.ReadAll(res.Body)
				require.NoError(t, errors.Join(err, res.Body.Close()))

				var jsonRes any
				require.NoError(t, json.Unmarshal(data, &jsonRes))
				require.Equal(t, testCase.expectResponse, jsonRes)
			}
		})
	}
}
