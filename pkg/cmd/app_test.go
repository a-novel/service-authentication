package cmdpkg_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/golib/ogen"
	otelpresets "github.com/a-novel/golib/otel/presets"
	"github.com/a-novel/golib/postgres"
	"github.com/a-novel/golib/smtp"

	"github.com/a-novel/service-authentication/migrations"
	"github.com/a-novel/service-authentication/models/config"
	"github.com/a-novel/service-authentication/pkg"
	cmdpkg "github.com/a-novel/service-authentication/pkg/cmd"
)

type TestConfig = config.App[*otelpresets.SentryOtelConfig, postgres.Config, *smtp.TestSender]

type AppTestSuite func(ctx context.Context, t *testing.T, config TestConfig)

func TestApp(t *testing.T) {
	testSuites := map[string]AppTestSuite{
		"Ping":           testAppPing,
		"AuthAnon":       testAppAuthAnon,
		"UserLifecycle":  testAppUserLifecycle,
		"RefreshToken":   testAppRefreshToken,
		"ResetPassword":  testAppResetPassword,
		"UpdateEmail":    testAppUpdateEmail,
		"UpdatePassword": testAppUpdatePassword,
		"UpdateRole":     testAppUpdateRole,
		"ListUsers":      testAppListUsers,
	}

	for testName, testSuite := range testSuites {
		t.Run(testName, func(t *testing.T) {
			postgres.RunIsolatedTransactionalTest(
				t, config.PostgresPresetTest, migrations.Migrations, func(ctx context.Context, t *testing.T) {
					t.Helper()

					port, err := ogen.GetRandomPort()
					require.NoError(t, err)

					appConfig := config.AppPresetTest(port)

					go func() {
						assert.NoError(t, cmdpkg.App(ctx, appConfig))
					}()

					security := pkg.NewBearerSource()
					client, err := pkg.NewAPIClient(
						ctx, fmt.Sprintf("http://localhost:%v/v1", appConfig.API.Port), security,
					)
					require.NoError(t, err)

					require.Eventually(t, func() bool {
						_, err = client.Ping(t.Context())

						return assert.NoError(t, err)
					}, 10*time.Second, 100*time.Millisecond)

					testSuite(ctx, t, appConfig)
				},
			)
		})
	}
}
