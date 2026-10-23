package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun/driver/pgdriver"

	postgrespresets "github.com/a-novel-kit/golib/postgres/presets"

	"github.com/a-novel/service-authentication/v2/internal/config"
)

func TestNewPostgresPreset(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		connection postgrespresets.Connection

		expectAddr       string
		expectUser       string
		expectPassword   string
		expectDatabase   string
		expectTLSEnabled bool
		expectPanic      string

		expectConnectorPanic bool
	}{
		{
			name: "Success",
			connection: postgrespresets.Connection{
				Host:       "postgres.internal",
				Port:       5433,
				User:       "authentication",
				Password:   "password-with-:/@%",
				Database:   "authentication",
				TLSEnabled: false,
			},
			expectAddr:       "postgres.internal:5433",
			expectUser:       "authentication",
			expectPassword:   "password-with-:/@%",
			expectDatabase:   "authentication",
			expectTLSEnabled: false,
		},
		{
			name: "Success/TLS",
			connection: postgrespresets.Connection{
				Host:       "postgres.internal",
				Port:       5432,
				User:       "authentication",
				Password:   "password",
				Database:   "authentication",
				TLSEnabled: true,
			},
			expectAddr:       "postgres.internal:5432",
			expectUser:       "authentication",
			expectPassword:   "password",
			expectDatabase:   "authentication",
			expectTLSEnabled: true,
		},
		{
			name: "Error/MissingPassword",
			connection: postgrespresets.Connection{
				Host:     "postgres.internal",
				Port:     5432,
				User:     "authentication",
				Database: "authentication",
			},
			expectPanic: "postgres password is empty",
		},
		{
			// An unset host never reaches the driver's default server.
			name: "Error/MissingHost",
			connection: postgrespresets.Connection{
				Port:     5432,
				User:     "authentication",
				Password: "password",
				Database: "authentication",
			},
			expectConnectorPanic: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if testCase.expectPanic != "" {
				require.PanicsWithValue(t, testCase.expectPanic, func() {
					config.NewPostgresPreset(testCase.connection, 12, 8)
				})

				return
			}

			preset := config.NewPostgresPreset(testCase.connection, 12, 8)

			if testCase.expectConnectorPanic {
				require.Panics(t, func() { pgdriver.NewConnector(preset.Options()...) })

				return
			}

			driverConfig := pgdriver.NewConnector(preset.Options()...).Config()

			require.Equal(t, testCase.expectAddr, driverConfig.Addr)
			require.Equal(t, testCase.expectUser, driverConfig.User)
			require.Equal(t, testCase.expectPassword, driverConfig.Password)
			require.Equal(t, testCase.expectDatabase, driverConfig.Database)
			require.Equal(t, testCase.expectTLSEnabled, driverConfig.TLSConfig != nil)
			require.Equal(t, 3*time.Minute, driverConfig.DialTimeout)
			require.Equal(t, 12, preset.MaxOpenConns)
			require.Equal(t, 8, preset.MaxIdleConns)
		})
	}
}
