package middlewares_test

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/jwt/v2"
	"github.com/a-novel-kit/jwt/v2/jwp"
	"github.com/a-novel-kit/jwt/v2/jws"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/handlers/middlewares"
	middlewaresmocks "github.com/a-novel/service-authentication/v2/internal/handlers/middlewares/mocks"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

func TestAuth(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type verifyClaimsMock struct {
		reqToken string
		resp     *session.Claims
		err      error
	}

	testCases := []struct {
		name string

		authHeader string

		permissions       []string
		permissionsByRole map[string][]string
		verifyClaimsMock  *verifyClaimsMock

		expectStatus int
		expectClaims *session.Claims
	}{
		{
			name: "Success",

			authHeader: "Bearer token",

			permissions: []string{"write"},
			permissionsByRole: map[string][]string{
				"role1": {"read", "write"},
				"role2": {"read"},
			},
			verifyClaimsMock: &verifyClaimsMock{
				reqToken: "token",
				resp: &session.Claims{
					UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:  []string{"role1"},
				},
			},

			expectStatus: http.StatusOK,
			expectClaims: &session.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				Roles:  []string{"role1"},
			},
		},
		{
			name: "Success/CompositePermissions",

			authHeader: "Bearer token",

			permissions: []string{"read", "write"},
			permissionsByRole: map[string][]string{
				"role1": {"read"},
				"role2": {"write"},
			},
			verifyClaimsMock: &verifyClaimsMock{
				reqToken: "token",
				resp: &session.Claims{
					UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:  []string{"role2"},
				},
			},

			expectStatus: http.StatusOK,
			expectClaims: &session.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				Roles:  []string{"role2"},
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
				resp: &session.Claims{
					UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:  []string{"role1"},
				},
			},

			expectStatus: http.StatusOK,
			expectClaims: &session.Claims{
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

			permissions: []string{"write"},
			permissionsByRole: map[string][]string{
				"role1": {"read", "write"},
				"role2": {"read"},
			},
			verifyClaimsMock: &verifyClaimsMock{
				reqToken: "token",
				resp: &session.Claims{
					UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:  []string{"role2"},
				},
			},

			expectStatus: http.StatusForbidden,
		},
		{
			// A token role the config does not define is an integrity fault, not a
			// permission denial: it returns 500, distinct from the 403 above, so it is
			// not lost among legitimate denials. This is the shape the 'user' default
			// produced.
			name: "Error/UnknownRole",

			authHeader: "Bearer token",

			permissions: []string{"write"},
			permissionsByRole: map[string][]string{
				"role1": {"read", "write"},
			},
			verifyClaimsMock: &verifyClaimsMock{
				reqToken: "token",
				resp: &session.Claims{
					UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:  []string{"ghost"},
				},
			},

			expectStatus: http.StatusInternalServerError,
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
			// Clients refresh their session on a 401; a 500 here strands an expired one.
			name: "Error/ExpiredToken",

			authHeader: "Bearer token",

			permissions: []string{"read", "write"},
			permissionsByRole: map[string][]string{
				"role1": {"read", "write"},
				"role2": {"read"},
			},
			verifyClaimsMock: &verifyClaimsMock{
				reqToken: "token",
				err:      fmt.Errorf("(ClaimsChecker.Unmarshal) %w: token expired", jwp.ErrInvalidClaims),
			},

			expectStatus: http.StatusUnauthorized,
		},
		{
			name: "Error/MalformedToken/Format",

			authHeader: "Bearer token",

			permissions: []string{"read", "write"},
			permissionsByRole: map[string][]string{
				"role1": {"read", "write"},
				"role2": {"read"},
			},
			verifyClaimsMock: &verifyClaimsMock{
				reqToken: "token",
				err:      fmt.Errorf("(HeaderDecoder.Decode) %w: expected at least 2 segments", jwt.ErrUnsupportedTokenFormat),
			},

			expectStatus: http.StatusUnauthorized,
		},
		{
			name: "Error/UnsupportedAlgorithm",

			authHeader: "Bearer token",

			permissions: []string{"read", "write"},
			permissionsByRole: map[string][]string{
				"role1": {"read", "write"},
				"role2": {"read"},
			},
			verifyClaimsMock: &verifyClaimsMock{
				reqToken: "token",
				err:      fmt.Errorf("(Recipient.Consume) %w: no compatible plugin found", jwt.ErrMismatchRecipientPlugin),
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
				verifyClaimsRequest := &servicejsonkeys.VerifyClaimsRequest{
					Usage:       servicejsonkeys.KeyUsageAuth,
					AccessToken: testCase.verifyClaimsMock.reqToken,
				}

				service.EXPECT().
					VerifyClaims(mock.Anything, verifyClaimsRequest).
					Return(testCase.verifyClaimsMock.resp, testCase.verifyClaimsMock.err)
			}

			middleware := middlewares.NewAuth(service, testCase.permissionsByRole, config.LoggerDev)
			w := httptest.NewRecorder()

			ctxClaims := new(*session.Claims)

			callback := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				claims, err := session.GetClaimsContext(r.Context())
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

	// A garbled token fails while the real verifier parses it, before any key is fetched, so an
	// unreachable JSON-keys server cannot turn the rejection into a server error.
	t.Run("Error/GarbledToken", func(t *testing.T) {
		t.Parallel()

		client, err := servicejsonkeys.NewClient(
			"127.0.0.1:1", grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		require.NoError(t, err)
		t.Cleanup(client.Close)

		verifier, err := servicejsonkeys.NewClaimsVerifier[session.Claims](client)
		require.NoError(t, err)

		handler := middlewares.NewAuth(verifier, map[string][]string{"role1": {"read"}}, config.LoggerDev).
			Middleware([]string{"read"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		garbledTokens := []string{
			"!!!.e30",
			base64.RawURLEncoding.EncodeToString([]byte("not json")) + ".e30",
		}

		for _, token := range garbledTokens {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer "+token)

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			require.Equal(t, http.StatusUnauthorized, w.Code, token)
		}
	})
}
