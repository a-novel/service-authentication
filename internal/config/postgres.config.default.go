package config

import (
	"net"
	"strconv"
	"time"

	"github.com/uptrace/bun/driver/pgdriver"

	postgrespresets "github.com/a-novel-kit/golib/postgres/presets"

	"github.com/a-novel/service-authentication/v2/internal/config/env"
)

const postgresDialTimeout = 3 * time.Minute

// PostgresConnection describes how to reach PostgreSQL. Host selects the discrete
// fields; an empty Host selects the legacy DSN fallback.
type PostgresConnection struct {
	DSN        string
	Host       string
	Port       int
	User       string
	Password   string
	Database   string
	TLSEnabled bool
}

// PostgresPresetDefault is the default PostgreSQL connection preset.
var PostgresPresetDefault = NewPostgresPreset(PostgresConnection{
	DSN:        env.PostgresDsn,
	Host:       env.PostgresHost,
	Port:       env.PostgresPort,
	User:       env.PostgresUser,
	Password:   env.PostgresPassword,
	Database:   env.PostgresDatabase,
	TLSEnabled: env.PostgresTLSEnabled,
}, env.PostgresMaxOpenConns, env.PostgresMaxIdleConns)

// NewPostgresPreset returns a PostgreSQL preset whose pool is bounded before it opens.
//
// Setting the limits on the handle afterwards stops working once anything has
// taken a connection, because the handle is cached; past that point they apply to
// nothing and report nothing.
func NewPostgresPreset(
	connection PostgresConnection,
	maxOpenConns int,
	maxIdleConns int,
) *postgrespresets.Default {
	options := append(connection.options(), pgdriver.WithDialTimeout(postgresDialTimeout))
	preset := postgrespresets.NewDefault(options...)
	preset.MaxOpenConns = maxOpenConns
	preset.MaxIdleConns = maxIdleConns

	return preset
}

func (connection PostgresConnection) options() []pgdriver.Option {
	if connection.Host == "" {
		return []pgdriver.Option{pgdriver.WithDSN(connection.DSN)}
	}

	if connection.Password == "" {
		panic("postgres password is empty")
	}

	return []pgdriver.Option{
		pgdriver.WithAddr(net.JoinHostPort(connection.Host, strconv.Itoa(connection.Port))),
		pgdriver.WithUser(connection.User),
		pgdriver.WithPassword(connection.Password),
		pgdriver.WithDatabase(connection.Database),
		pgdriver.WithInsecure(!connection.TLSEnabled),
	}
}
