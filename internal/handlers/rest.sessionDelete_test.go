package handlers_test

import (
	"cmp"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/config/configtest"
	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-authentication/v2/internal/handlers/mocks"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

func TestRestSessionDelete(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	sessionID := uuid.MustParse("00000000-0000-0000-0000-0000000000a2")
	claims := &session.Claims{UserID: new(userID), RefreshTokenID: "00000000-0000-0000-0000-0000000000a1"}
	validPath := "/v2/sessions/" + sessionID.String()

	type serviceMock struct {
		err error
	}

	testCases := []struct {
		name string

		// path replaces validPath when set.
		path string
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

			serviceMock: &serviceMock{err: core.ErrSessionNotFound},

			expectStatus: http.StatusNotFound,
		},
		{
			// An active session is signed out first, then removed.
			name: "Error/Active",

			serviceMock: &serviceMock{err: core.ErrSessionActive},

			expectStatus: http.StatusConflict,
		},
		{
			name: "Error/Internal",

			serviceMock: &serviceMock{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
		{
			name: "Error/MalformedSessionID",

			path: "/v2/sessions/session",

			expectStatus: http.StatusBadRequest,
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

			service := handlersmocks.NewMockRestSessionDeleteService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.SessionDeleteRequest{UserID: userID, ID: sessionID}).
					Return(testCase.serviceMock.err)
			}

			ctx := session.SetClaimsContext(t.Context(), claims)
			if testCase.noClaims {
				ctx = t.Context()
			}

			// The router reads the session ID from the path.
			router := chi.NewRouter()
			router.Delete("/v2/sessions/{sessionID}", handlers.NewRestSessionDelete(service, configtest.Logger).ServeHTTP)

			w := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(ctx, http.MethodDelete, cmp.Or(testCase.path, validPath), nil)

			router.ServeHTTP(w, request)

			res := w.Result()
			require.NoError(t, res.Body.Close())
			require.Equal(t, testCase.expectStatus, res.StatusCode)
		})
	}
}
