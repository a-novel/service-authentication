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
	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"
)

//go:embed testdata/client/main.go
var clientProgram []byte

func TestPermissionsHandler(t *testing.T) {
	t.Parallel()

	// grandchild -> child -> parent. Each level adds one permission.
	inheritance := map[string]authconfig.Role{
		"parent":     {Permissions: []string{"parent:read"}},
		"child":      {Inherits: []string{"parent"}, Permissions: []string{"child:read"}},
		"grandchild": {Inherits: []string{"child"}, Permissions: []string{"grandchild:read"}},
	}

	// NewAuthHandler resolves role inheritance transitively at startup. A role grants every
	// permission its ancestors do, and only those.
	testCases := []struct {
		name string

		roles      map[string]authconfig.Role
		role       string
		permission string

		expectStatus int
		expectPanic  bool
	}{
		{
			name: "Success/OwnPermission",

			roles:      inheritance,
			role:       "grandchild",
			permission: "grandchild:read",

			expectStatus: http.StatusOK,
		},
		{
			name: "Success/ParentPermission",

			roles:      inheritance,
			role:       "grandchild",
			permission: "child:read",

			expectStatus: http.StatusOK,
		},
		{
			name: "Success/GrandparentPermission",

			roles:      inheritance,
			role:       "grandchild",
			permission: "parent:read",

			expectStatus: http.StatusOK,
		},
		{
			// A role never gains a permission defined on a role that inherits it.
			name: "Error/DescendantPermission",

			roles:      inheritance,
			role:       "parent",
			permission: "grandchild:read",

			expectStatus: http.StatusForbidden,
		},
		{
			// Circular inheritance is a programming error, reported at startup.
			name: "Error/CircularInheritance",

			roles: map[string]authconfig.Role{
				"a": {Inherits: []string{"b"}},
				"b": {Inherits: []string{"a"}},
			},

			expectPanic: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			permissions := serviceauthentication.Permissions{Roles: testCase.roles}
			verifier := fakeVerifier{roles: []string{testCase.role}}

			if testCase.expectPanic {
				require.Panics(t, func() { serviceauthentication.NewAuthHandler(verifier, permissions, config.LoggerDev) })

				return
			}

			handler := serviceauthentication.NewAuthHandler(verifier, permissions, config.LoggerDev)

			router := chi.NewRouter()
			handler(router, testCase.permission).Get("/", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer token")

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			require.Equal(t, testCase.expectStatus, rec.Code)
		})
	}

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
					testCase.prefix+"SMTP_MAX_CONCURRENT=invalid",
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
) (*session.Claims, error) {
	return &session.Claims{Roles: f.roles}, nil
}
