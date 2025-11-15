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

	"github.com/a-novel/golib/postgres"
	jkpkg "github.com/a-novel/service-json-keys/v2/pkg"

	"github.com/a-novel/service-authentication/internal/config"
	"github.com/a-novel/service-authentication/internal/handlers"
	handlersmocks "github.com/a-novel/service-authentication/internal/handlers/mocks"
)

func TestHealth(t *testing.T) {
	t.Parallel()

	type healthClientSmtpMock struct {
		err error
	}

	type healthApiJsonKeysMock struct {
		res *jkpkg.StatusResponse
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

			request: httptest.NewRequest(http.MethodPost, "/", nil),

			healthClientSmtpMock: &healthClientSmtpMock{},
			healthApiJsonKeysMock: &healthApiJsonKeysMock{
				res: new(jkpkg.StatusResponse),
			},

			expectResponse: map[string]any{
				"client:postgres": map[string]any{
					"status": handlers.HealthStatusUp,
				},
				"client:smtp": map[string]any{
					"status": handlers.HealthStatusUp,
				},
				"api:jsonKeys": map[string]any{
					"status": handlers.HealthStatusUp,
				},
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "Error",

			request: httptest.NewRequest(http.MethodPost, "/", nil),

			healthClientSmtpMock: &healthClientSmtpMock{
				err: errors.New("error smtp"),
			},
			healthApiJsonKeysMock: &healthApiJsonKeysMock{
				err: errors.New("error json keys"),
			},

			expectResponse: map[string]any{
				"client:postgres": map[string]any{
					"status": handlers.HealthStatusUp,
				},
				"client:smtp": map[string]any{
					"status": handlers.HealthStatusDown,
					"err":    "error smtp",
				},
				"api:jsonKeys": map[string]any{
					"status": handlers.HealthStatusDown,
					"err":    "error json keys",
				},
			},
			expectStatus: http.StatusOK,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			healthClientSmtp := handlersmocks.NewMockHealthClientSmtp(t)
			healthApiJsonKeys := handlersmocks.NewMockHealthApiJsonkeys(t)

			if testCase.healthClientSmtpMock != nil {
				healthClientSmtp.EXPECT().
					Ping().
					Return(testCase.healthClientSmtpMock.err)
			}

			if testCase.healthApiJsonKeysMock != nil {
				healthApiJsonKeys.EXPECT().
					Status(mock.Anything, new(jkpkg.StatusRequest)).
					Return(testCase.healthApiJsonKeysMock.res, testCase.healthApiJsonKeysMock.err)
			}

			handler := handlers.NewHealth(healthApiJsonKeys, healthClientSmtp)
			w := httptest.NewRecorder()

			rCtx := testCase.request.Context()
			rCtx, err := postgres.NewContext(rCtx, config.PostgresPresetTest)
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
