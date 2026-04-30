package handlers_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-authentication/v2/internal/config/configtest"
	"github.com/a-novel/service-authentication/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-authentication/v2/internal/handlers/mocks"
)

func TestHealth(t *testing.T) {
	t.Parallel()

	type healthClientSmtpMock struct {
		err error
	}

	type healthApiJsonKeysMock struct {
		res *servicejsonkeys.StatusResponse
		err error
	}

	testCases := []struct {
		name string

		request *http.Request

		healthClientSmtpMock  *healthClientSmtpMock
		healthApiJsonKeysMock *healthApiJsonKeysMock

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", nil),

			healthClientSmtpMock: &healthClientSmtpMock{},
			healthApiJsonKeysMock: &healthApiJsonKeysMock{
				res: new(servicejsonkeys.StatusResponse),
			},

			expectResponse: map[string]any{
				"client:postgres": map[string]any{
					"status": handlers.RestHealthStatusUp,
				},
				"client:smtp": map[string]any{
					"status": handlers.RestHealthStatusUp,
				},
				"api:jsonKeys": map[string]any{
					"status": handlers.RestHealthStatusUp,
				},
			},
			expectStatus: http.StatusOK,
		},
		{
			// The SMTP and json-keys dependency mocks return errors; the response must
			// report status=down for those dependencies WITHOUT echoing the underlying
			// error string. The exact-match assertion on expectResponse below is the
			// regression guard: it fails if any extra field (notably a re-introduced
			// "err") leaks into the public response shape.
			name: "Error",

			request: httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", nil),

			healthClientSmtpMock: &healthClientSmtpMock{
				err: errors.New("error smtp"),
			},
			healthApiJsonKeysMock: &healthApiJsonKeysMock{
				err: errors.New("error json keys"),
			},

			expectResponse: map[string]any{
				"client:postgres": map[string]any{
					"status": handlers.RestHealthStatusUp,
				},
				"client:smtp": map[string]any{
					"status": handlers.RestHealthStatusDown,
				},
				"api:jsonKeys": map[string]any{
					"status": handlers.RestHealthStatusDown,
				},
			},
			expectStatus: http.StatusOK,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			healthClientSmtp := handlersmocks.NewMockRestHealthClientSmtp(t)
			healthApiJsonKeys := handlersmocks.NewMockRestHealthApiJsonKeys(t)

			if testCase.healthClientSmtpMock != nil {
				healthClientSmtp.EXPECT().
					Ping().
					Return(testCase.healthClientSmtpMock.err)
			}

			if testCase.healthApiJsonKeysMock != nil {
				healthApiJsonKeys.EXPECT().
					Status(mock.Anything, new(servicejsonkeys.StatusRequest)).
					Return(testCase.healthApiJsonKeysMock.res, testCase.healthApiJsonKeysMock.err)
			}

			handler := handlers.NewRestHealth(healthApiJsonKeys, healthClientSmtp)
			w := httptest.NewRecorder()

			rCtx := testCase.request.Context()
			rCtx, err := postgres.NewContext(rCtx, configtest.PostgresPreset)
			require.NoError(t, err)

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
