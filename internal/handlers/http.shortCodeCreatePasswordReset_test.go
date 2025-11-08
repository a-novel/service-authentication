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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/internal/handlers"
	handlersmocks "github.com/a-novel/service-authentication/internal/handlers/mocks"
	"github.com/a-novel/service-authentication/internal/services"
)

func TestShortCodeCreatePasswordReset(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceMock struct {
		req  *services.ShortCodeCreatePasswordResetRequest
		resp *services.ShortCode
		err  error
	}

	testCases := []struct {
		name string

		request *http.Request

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

			serviceMock: &serviceMock{
				req: &services.ShortCodeCreatePasswordResetRequest{
					Email: "new_user@provider.com",
					Lang:  "fr",
				},
				resp: &services.ShortCode{
					ID:        uuid.MustParse("00000000-0000-0000-0000-111111111111"),
					Usage:     services.ShortCodeUsageResetPassword,
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

			serviceMock: &serviceMock{
				req: &services.ShortCodeCreatePasswordResetRequest{
					Email: "new_user@provider.com",
					Lang:  "fr",
				},
				err: errFoo,
			},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockShortCodeCreatePasswordResetService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, testCase.serviceMock.req).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewShortCodeCreatePasswordReset(service)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, testCase.request)

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
