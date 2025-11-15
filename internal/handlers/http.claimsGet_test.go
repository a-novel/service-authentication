package handlers_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/handlers"
	"github.com/a-novel/service-authentication/v2/internal/handlers/middlewares"
	"github.com/a-novel/service-authentication/v2/internal/services"
)

func TestClaimsGet(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		request *http.Request
		claims  *services.AccessTokenClaims

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: httptest.NewRequest(http.MethodPost, "/", nil),
			claims: &services.AccessTokenClaims{
				UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				Roles:          []string{"user"},
				RefreshTokenID: "refreshToken",
			},

			expectResponse: map[string]any{
				"userID":         "00000000-0000-0000-0000-000000000001",
				"roles":          []any{"user"},
				"refreshTokenID": "refreshToken",
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "NoClaims",

			request: httptest.NewRequest(http.MethodPost, "/", nil),

			expectStatus: http.StatusForbidden,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			handler := handlers.NewClaimsGet()
			w := httptest.NewRecorder()

			rCtx := testCase.request.Context()
			rCtx = middlewares.SetClaimsContext(rCtx, testCase.claims)

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
