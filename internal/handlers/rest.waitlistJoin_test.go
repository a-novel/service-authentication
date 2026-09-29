package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-authentication/v2/internal/handlers/mocks"
)

func TestRESTWaitlistJoin(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		body   string
		err    error
		status int
		retry  string
	}{
		{name: "Accepted", status: http.StatusAccepted},
		{name: "MalformedJSON", body: "{", status: http.StatusBadRequest},
		{name: "InvalidRequest", err: core.ErrInvalidRequest, status: http.StatusUnprocessableEntity},
		{name: "Busy", err: dao.ErrWaitlistBusy, status: http.StatusTooManyRequests, retry: "60"},
		{name: "Unavailable", err: dao.ErrWaitlistUnavailable, status: http.StatusServiceUnavailable, retry: "60"},
		{name: "UnexpectedFailure", err: errors.New("database unavailable"), status: http.StatusInternalServerError},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRESTWaitlistJoinService(t)

			body := testCase.body
			if body == "" {
				body = `{"email":"member@example.com","lang":"en"}`

				service.EXPECT().Exec(mock.Anything, &core.WaitlistJoinRequest{Email: "member@example.com", Lang: "en"}).
					Return(testCase.err).Once()
			}

			request := httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/v2/waitlist", strings.NewReader(body))
			response := httptest.NewRecorder()
			handlers.NewRESTWaitlistJoin(service, config.LoggerDev).ServeHTTP(response, request)
			require.Equal(t, testCase.status, response.Code)
			require.Equal(t, testCase.retry, response.Header().Get("Retry-After"))

			if testCase.status == http.StatusAccepted {
				require.Empty(t, response.Body.String())
			}
		})
	}
}
