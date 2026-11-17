package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/config/configtest"
	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-authentication/v2/internal/handlers/mocks"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

func TestRestShortCodeCancelEmailUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	claims := &session.Claims{UserID: new(userID), RefreshTokenID: "00000000-0000-0000-0000-0000000000a1"}

	type serviceMock struct {
		err error
	}

	testCases := []struct {
		name string

		// noClaims sends the request without the claims the auth middleware sets.
		noClaims bool

		serviceMock *serviceMock

		expectStatus int
	}{
		{
			name: "Success",

			serviceMock: &serviceMock{},

			expectStatus: http.StatusNoContent,
		},
		{
			name: "Error/NotFound",

			serviceMock: &serviceMock{err: core.ErrShortCodeNotFound},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/Internal",

			serviceMock: &serviceMock{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
		{
			name: "Error/MissingClaims",

			noClaims: true,

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestShortCodeCancelEmailUpdateService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.ShortCodeCancelEmailUpdateRequest{UserID: userID}).
					Return(testCase.serviceMock.err)
			}

			ctx := session.SetClaimsContext(t.Context(), claims)
			if testCase.noClaims {
				ctx = t.Context()
			}

			handler := handlers.NewRestShortCodeCancelEmailUpdate(service, configtest.Logger)
			w := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/v2/short-code/update-email", nil)

			handler.ServeHTTP(w, request)

			res := w.Result()
			require.NoError(t, res.Body.Close())
			require.Equal(t, testCase.expectStatus, res.StatusCode)
		})
	}
}
