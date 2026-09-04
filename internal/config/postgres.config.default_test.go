package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun/driver/pgdriver"

	"github.com/a-novel/service-authentication/v2/internal/config"
)

func TestPostgresConnection(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		connection config.PostgresConnection

		expectAddr       string
		expectUser       string
		expectPassword   string
		expectDatabase   string
		expectTLSEnabled bool
		expectPanic      string
	}{
		{
			name: "Success/Discrete",
			connection: config.PostgresConnection{
				DSN:        "not-used",
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
			name: "Success/DiscreteTLS",
			connection: config.PostgresConnection{
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
			name: "Success/LegacyDSN",
			connection: config.PostgresConnection{
				DSN: "postgres://legacy-user@legacy.internal:6432/legacy-db?sslmode=disable",
			},
			expectAddr:       "legacy.internal:6432",
			expectUser:       "legacy-user",
			expectPassword:   "",
			expectDatabase:   "legacy-db",
			expectTLSEnabled: false,
		},
		{
			name: "Error/MissingDiscretePassword",
			connection: config.PostgresConnection{
				Host:     "postgres.internal",
				Port:     5432,
				User:     "authentication",
				Database: "authentication",
			},
			expectPanic: "postgres password is empty",
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
