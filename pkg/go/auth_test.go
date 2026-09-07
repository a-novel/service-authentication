package serviceauthentication_test

import (
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	servicejsonkeys "github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/core"
	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"
)

//go:embed testdata/client/main.go
var clientProgram []byte

func TestPermissionsHandler(t *testing.T) {
	t.Parallel()

	t.Run("Success/ImportIgnoresServerEnvironment", func(t *testing.T) {
		t.Parallel()

		moduleRoot, err := filepath.Abs("../..")
		require.NoError(t, err)

		consumerDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(consumerDir, "go.mod"), []byte(fmt.Sprintf(`module test/client

go 1.27.1

require github.com/a-novel/service-authentication/v2 v2.0.0

replace github.com/a-novel/service-authentication/v2 => %s
`, filepath.ToSlash(moduleRoot))), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(consumerDir, "main.go"), clientProgram, 0o600))

		consumerPath := filepath.Join(consumerDir, "consumer")
		build := exec.CommandContext(t.Context(), "go", "build", "-mod=mod", "-o", "consumer", ".")
		build.Dir = consumerDir

		build.Env = append(os.Environ(), "GOWORK=off")
		output, err := build.CombinedOutput()
		require.NoError(t, err, string(output))

		testCases := []struct {
			name   string
			prefix string
		}{
			{name: "Unprefixed"},
			{name: "Prefixed", prefix: "AUTH_CLIENT_TEST_"},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				t.Parallel()

				command := exec.CommandContext(t.Context(), consumerPath)
				command.Dir = consumerDir

				command.Env = append(os.Environ(),
					"SERVICE_AUTHENTICATION_ENV_PREFIX="+testCase.prefix,
					"SERVICE_JSON_KEYS_ENV_PREFIX="+testCase.prefix,
					testCase.prefix+"REST_TIMEOUT_READ=invalid",
					testCase.prefix+"SMTP_TIMEOUT=invalid",
					testCase.prefix+"POSTGRES_PORT=invalid",
					testCase.prefix+"GRPC_PING=invalid",
					testCase.prefix+"OTEL=invalid",
				)

				output, err := command.CombinedOutput()
				require.NoError(t, err, string(output))
			})
		}
	})

	t.Run("Success/DependencyBoundary", func(t *testing.T) {
		t.Parallel()

		command := exec.CommandContext(t.Context(), "go", "list", "-deps", ".")

		command.Env = append(os.Environ(), "GOWORK=off")
		output, err := command.Output()
		require.NoError(t, err)

		dependencies := strings.Split(strings.TrimSpace(string(output)), "\n")
		testCases := []struct {
			name       string
			configPath string
		}{
			{name: "Authentication", configPath: "github.com/a-novel/service-authentication/v2/internal/config"},
			{name: "JsonKeys", configPath: "github.com/a-novel/service-json-keys/v2/internal/config"},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				t.Parallel()

				require.NotContains(t, dependencies, testCase.configPath)
				require.NotContains(t, dependencies, testCase.configPath+"/env")
			})
		}
	})
}

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
