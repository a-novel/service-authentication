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

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-authentication/v2/internal/handlers/mocks"
	"github.com/a-novel/service-authentication/v2/internal/services"
)

func TestShortCodeCreateRegister(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceMock struct {
		req  *services.ShortCodeCreateRegisterRequest
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
				req: &services.ShortCodeCreateRegisterRequest{
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

			expectStatus: http.StatusAccepted,
		},
		{
			name: "Success/AlreadyExists",

			request: httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{
				"email": "existing_user@provider.com",
				"lang": "fr"
			}`)),

			serviceMock: &serviceMock{
				req: &services.ShortCodeCreateRegisterRequest{
					Email: "existing_user@provider.com",
					Lang:  "fr",
				},
				err: dao.ErrCredentialsInsertAlreadyExists,
			},

			// Returns 202 to prevent email enumeration.
			expectStatus: http.StatusAccepted,
		},
		{
			name: "Error/Internal",

			request: httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{
				"email": "new_user@provider.com",
				"lang": "fr"
			}`)),

			serviceMock: &serviceMock{
				req: &services.ShortCodeCreateRegisterRequest{
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

			service := handlersmocks.NewMockShortCodeCreateRegisterService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, testCase.serviceMock.req).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewShortCodeCreateRegister(service, config.LoggerDev)
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
