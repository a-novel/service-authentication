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

func TestRestCredentialsUpdateEmail(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	validBody := `{"userID": "` + credentialsFixture.ID.String() + `", "shortCode": "abcdef", "lang": "fr"}`
	serviceRequest := &core.CredentialsUpdateEmailRequest{UserID: credentialsFixture.ID, ShortCode: "abcdef", Lang: "fr"}

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
			name: "Error/CredentialsNotFound",

			serviceMock: &serviceMock{err: core.ErrCredentialsNotFound},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/CredentialsAlreadyExists",

			serviceMock: &serviceMock{err: core.ErrCredentialsAlreadyExists},

			expectStatus: http.StatusConflict,
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
			name: "Error/MailDeliveryUnavailable",

			serviceMock: &serviceMock{err: errors.Join(errFoo, core.ErrMailDeliveryUnavailable)},

			expectStatus: http.StatusServiceUnavailable,
		},
		{
			name: "Error/Internal",

			serviceMock: &serviceMock{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
		{
			name: "Error/MalformedBody",

			body: `{"userID": "user"}`,

			expectStatus: http.StatusBadRequest,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestCredentialsUpdateEmailService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, serviceRequest).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewRestCredentialsUpdateEmail(service, config.LoggerDev)
			w := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(
				t.Context(), http.MethodPatch, "/v2/credentials/email", strings.NewReader(cmp.Or(testCase.body, validBody)),
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
