package handlers_test

import (
	"cmp"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-authentication/v2/internal/handlers/mocks"
)

func TestRestWaitlistJoin(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	validBody := `{"email": "member@example.com", "lang": "en"}`
	serviceRequest := &core.WaitlistJoinRequest{Email: "member@example.com", Lang: "en"}

	type serviceMock struct {
		err error
	}

	testCases := []struct {
		name string

		// body replaces validBody when set.
		body string

		serviceMock *serviceMock

		expectStatus     int
		expectRetryAfter string
	}{
		{
			name: "Success",

			serviceMock: &serviceMock{},

			expectStatus: http.StatusAccepted,
		},
		{
			name: "Error/InvalidRequest",

			serviceMock: &serviceMock{err: core.ErrInvalidRequest},

			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "Error/Busy",

			serviceMock: &serviceMock{err: core.ErrWaitlistBusy},

			expectStatus:     http.StatusTooManyRequests,
			expectRetryAfter: "60",
		},
		{
			name: "Error/Unavailable",

			serviceMock: &serviceMock{err: core.ErrWaitlistUnavailable},

			expectStatus:     http.StatusServiceUnavailable,
			expectRetryAfter: "60",
		},
		{
			name: "Error/Internal",

			serviceMock: &serviceMock{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
		{
			name: "Error/MalformedBody",

			body: `{"email": `,

			expectStatus: http.StatusBadRequest,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestWaitlistJoinService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, serviceRequest).
					Return(testCase.serviceMock.err)
			}

			handler := handlers.NewRestWaitlistJoin(service, config.LoggerDev)
			w := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(
				t.Context(), http.MethodPut, "/v2/waitlist", strings.NewReader(cmp.Or(testCase.body, validBody)),
			)

			handler.ServeHTTP(w, request)

			res := w.Result()
			require.Equal(t, testCase.expectStatus, res.StatusCode)
			require.Equal(t, testCase.expectRetryAfter, res.Header.Get("Retry-After"))

			if testCase.expectStatus == http.StatusAccepted {
				require.Empty(t, w.Body.String())
			}

			require.NoError(t, res.Body.Close())
		})
	}
}
