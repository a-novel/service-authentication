package serviceauthentication_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	servicejsonkeys "github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/core"
	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"
)

// fakeVerifier stands in for the JSON-keys claims verifier: it returns fixed claims for any
// token, so the test exercises NewAuthHandler's role resolution without a running service.
type fakeVerifier struct {
	roles []string
}

func (f fakeVerifier) VerifyClaims(
	_ context.Context, _ *servicejsonkeys.VerifyClaimsRequest,
) (*core.AccessTokenClaims, error) {
	return &core.AccessTokenClaims{Roles: f.roles}, nil
}

// NewAuthHandler resolves role inheritance transitively at startup and wraps it in lo.Must.
// A role must grant every permission its ancestors do, and only those — the piece with real
// logic in this package, and the one service-narrative-engine is about to mount routes against.
func TestNewAuthHandlerResolvesInheritance(t *testing.T) {
	t.Parallel()

	// grandchild -> child -> parent. Each level adds one permission; grandchild must end up
	// with all three.
	permissions := serviceauthentication.Permissions{
		Roles: map[string]config.Role{
			"parent":     {Permissions: []string{"parent:read"}},
			"child":      {Inherits: []string{"parent"}, Permissions: []string{"child:read"}},
			"grandchild": {Inherits: []string{"child"}, Permissions: []string{"grandchild:read"}},
		},
	}

	// gatedStatus mounts a route requiring `permission`, gates it with a token bearing `role`,
	// and returns the status a request receives.
	gatedStatus := func(t *testing.T, role, permission string) int {
		t.Helper()

		handler := serviceauthentication.NewAuthHandler(
			fakeVerifier{roles: []string{role}}, permissions, config.LoggerDev,
		)

		router := chi.NewRouter()
		handler(router, permission).Get("/", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer token")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		return rec.Code
	}

	t.Run("a role reaches an inherited permission", func(t *testing.T) {
		t.Parallel()

		// grandchild transitively inherits parent:read, two levels up.
		require.Equal(t, http.StatusOK, gatedStatus(t, "grandchild", "parent:read"))
		// and the intermediate one.
		require.Equal(t, http.StatusOK, gatedStatus(t, "grandchild", "child:read"))
		// and its own.
		require.Equal(t, http.StatusOK, gatedStatus(t, "grandchild", "grandchild:read"))
	})

	t.Run("inheritance does not flow downward", func(t *testing.T) {
		t.Parallel()

		// parent must not gain a permission defined on a role that inherits it.
		require.Equal(t, http.StatusForbidden, gatedStatus(t, "parent", "grandchild:read"))
	})
}

// A permission map whose inheritance cannot be resolved — a role inheriting itself — is a
// boot-time programming error, and NewAuthHandler's lo.Must turns it into a panic rather than a
// silently empty permission set.
func TestNewAuthHandlerPanicsOnCircularInheritance(t *testing.T) {
	t.Parallel()

	permissions := serviceauthentication.Permissions{
		Roles: map[string]config.Role{
			"a": {Inherits: []string{"b"}},
			"b": {Inherits: []string{"a"}},
		},
	}

	require.Panics(t, func() {
		serviceauthentication.NewAuthHandler(fakeVerifier{}, permissions, config.LoggerDev)
	})
}
