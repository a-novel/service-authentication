package middlewares_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	jkmodels "github.com/a-novel/service-json-keys/models"
	"github.com/a-novel/service-json-keys/pkg"

	"github.com/a-novel-kit/jwt/jws"

	"github.com/a-novel/service-authentication/internal/handlers/middlewares"
	middlewaresmocks "github.com/a-novel/service-authentication/internal/handlers/middlewares/mocks"
	"github.com/a-novel/service-authentication/internal/services"
)

func TestAuth(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type verifyClaimsMock struct {
		reqToken string
		resp     *services.AccessTokenClaims
		err      error
	}

	testCases := []struct {
		name string

		authHeader string

		permissions       []string
		permissionsByRole map[string][]string
		verifyClaimsMock  *verifyClaimsMock

		expectStatus int
		expectClaims *services.AccessTokenClaims
	}{
		{
			name: "Success",

			authHeader: "Bearer token",

			permissions: []string{"read", "write"},
			permissionsByRole: map[string][]string{
				"role1": {"read", "write"},
				"role2": {"read"},
			},
			verifyClaimsMock: &verifyClaimsMock{
				reqToken: "token",
				resp: &services.AccessTokenClaims{
					UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:  []string{"role1"},
				},
			},

			expectStatus: http.StatusOK,
			expectClaims: &services.AccessTokenClaims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				Roles:  []string{"role1"},
			},
		},
		{
			name: "Success/CompositeRole",

			authHeader: "Bearer token",

			permissions: []string{"read", "write"},
			permissionsByRole: map[string][]string{
				"role1": {"read"},
				"role2": {"write"},
			},
			verifyClaimsMock: &verifyClaimsMock{
				reqToken: "token",
				resp: &services.AccessTokenClaims{
					UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:  []string{"role1", "role2"},
				},
			},

			expectStatus: http.StatusOK,
			expectClaims: &services.AccessTokenClaims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				Roles:  []string{"role1", "role2"},
			},
		},
		{
			name: "Success/NoPermissionsRequired",

			authHeader: "Bearer token",

			permissionsByRole: map[string][]string{
				"role1": {"read", "write"},
				"role2": {"read"},
			},
			verifyClaimsMock: &verifyClaimsMock{
				reqToken: "token",
				resp: &services.AccessTokenClaims{
					UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:  []string{"role1"},
				},
			},

			expectStatus: http.StatusOK,
			expectClaims: &services.AccessTokenClaims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				Roles:  []string{"role1"},
			},
		},
		{
			name: "Success/NoTokenButNoPermissionsRequired",

			permissionsByRole: map[string][]string{
				"role1": {"read", "write"},
				"role2": {"read"},
			},

			expectStatus: http.StatusOK,
		},

		{
			name: "Error/NoToken",

			permissions: []string{"read", "write"},
			permissionsByRole: map[string][]string{
				"role1": {"read", "write"},
				"role2": {"read"},
			},

			expectStatus: http.StatusUnauthorized,
		},
		{
			name: "Error/MalformedToken/NoBearer",

			authHeader: "token",

			permissions: []string{"read", "write"},
			permissionsByRole: map[string][]string{
				"role1": {"read", "write"},
				"role2": {"read"},
			},

			expectStatus: http.StatusUnauthorized,
		},
		{
			name: "Error/MalformedToken/TooMuchParts",

			authHeader: "Bearer tok en",

			permissions: []string{"read", "write"},
			permissionsByRole: map[string][]string{
				"role1": {"read", "write"},
				"role2": {"read"},
			},

			expectStatus: http.StatusUnauthorized,
		},
		{
			name: "Error/MalformedToken/BearerOnly",

			authHeader: "Bearer",

			permissions: []string{"read", "write"},
			permissionsByRole: map[string][]string{
				"role1": {"read", "write"},
				"role2": {"read"},
			},

			expectStatus: http.StatusUnauthorized,
		},
		{
			name: "Error/LackAuthorization",

			authHeader: "Bearer token",

			permissions: []string{"read", "write"},
			permissionsByRole: map[string][]string{
				"role1": {"read", "write"},
				"role2": {"read"},
			},
			verifyClaimsMock: &verifyClaimsMock{
				reqToken: "token",
				resp: &services.AccessTokenClaims{
					UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:  []string{"role2"},
				},
			},

			expectStatus: http.StatusUnauthorized,
		},
		{
			name: "Error/InvalidSignature",

			authHeader: "Bearer token",

			permissions: []string{"read", "write"},
			permissionsByRole: map[string][]string{
				"role1": {"read", "write"},
				"role2": {"read"},
			},
			verifyClaimsMock: &verifyClaimsMock{
				reqToken: "token",
				err:      jws.ErrInvalidSignature,
			},

			expectStatus: http.StatusUnauthorized,
		},
		{
			name: "Error/VerifyClaimsError",

			authHeader: "Bearer token",

			permissions: []string{"read", "write"},
			permissionsByRole: map[string][]string{
				"role1": {"read", "write"},
				"role2": {"read"},
			},
			verifyClaimsMock: &verifyClaimsMock{
				reqToken: "token",
				err:      errFoo,
			},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := middlewaresmocks.NewMockAuthClaimsVerifier(t)

			if testCase.verifyClaimsMock != nil {
				service.EXPECT().
					VerifyClaims(
						mock.Anything, jkmodels.KeyUsageAuth,
						testCase.verifyClaimsMock.reqToken,
						(*pkg.VerifyClaimsOptions)(nil),
					).
					Return(testCase.verifyClaimsMock.resp, testCase.verifyClaimsMock.err)
			}

			middleware := middlewares.NewAuth(service, testCase.permissionsByRole)
			w := httptest.NewRecorder()

			ctxClaims := new(*services.AccessTokenClaims)

			callback := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				claims, err := middlewares.GetClaimsContext(r.Context())
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)

					return
				}

				*ctxClaims = claims

				w.WriteHeader(http.StatusOK)
			})
			handler := middleware.Middleware(testCase.permissions)(callback)

			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
			if testCase.authHeader != "" {
				req.Header.Set("Authorization", testCase.authHeader)
			}

			handler.ServeHTTP(w, req)

			res := w.Result()

			require.Equal(t, testCase.expectStatus, res.StatusCode)
			require.Equal(t, testCase.expectClaims, *ctxClaims)
		})
	}
}
