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

func TestRestCredentialsResetPassword(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	validBody := `{"password": "Louvre", "shortCode": "abcdef", "userID": "` + credentialsFixture.ID.String() + `"}`
	serviceRequest := &core.CredentialsUpdatePasswordRequest{
		Password: "Louvre", ShortCode: "abcdef", UserID: credentialsFixture.ID,
	}

	type serviceMock struct {
		resp *core.Credentials
		err  error
	}

	testCases := []struct {
		name string

		// body replaces validBody when set.
		body string

		serviceMock *serviceMock

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			serviceMock: &serviceMock{resp: credentialsFixture},

			expectStatus:   http.StatusOK,
			expectResponse: credentialsFixtureJSON,
		},
		{
			// An unknown account answers as a wrong code does, so a caller cannot probe accounts.
			name: "Error/CredentialsNotFound",

			serviceMock: &serviceMock{err: core.ErrCredentialsNotFound},

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/ShortCodeInvalid",

			serviceMock: &serviceMock{err: core.ErrShortCodeInvalid},

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/InvalidRequest",

			serviceMock: &serviceMock{err: core.ErrInvalidRequest},

			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "Error/Internal",

			serviceMock: &serviceMock{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
		{
			name: "Error/MalformedBody",

			body: `{"password": `,

			expectStatus: http.StatusBadRequest,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestCredentialsResetPasswordService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, serviceRequest).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewRestCredentialsResetPassword(service, config.LoggerDev)
			w := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(
				t.Context(), http.MethodPut, "/v2/credentials/password", strings.NewReader(cmp.Or(testCase.body, validBody)),
			)

			handler.ServeHTTP(w, request)

			res := w.Result()
			require.Equal(t, testCase.expectStatus, res.StatusCode)

			if testCase.expectResponse != nil {
				requireJSONBody(t, res, testCase.expectResponse)
			}

			require.NoError(t, res.Body.Close())
		})
	}
}
