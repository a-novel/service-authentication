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

// postgresConnection reaches PostgreSQL through the POSTGRES_* variables.
var postgresConnection = PostgresConnection{
	DSN:        env.PostgresDSN,
	Host:       env.PostgresHost,
	Port:       env.PostgresPort,
	User:       env.PostgresUser,
	Password:   env.PostgresPassword,
	Database:   env.PostgresDatabase,
	TLSEnabled: env.PostgresTLSEnabled,
}

// PostgresPresetDefault is the default PostgreSQL connection preset.
var PostgresPresetDefault = NewPostgresPreset(postgresConnection, env.PostgresMaxOpenConns, env.PostgresMaxIdleConns)

// PostgresPresetMigrations is the migrations job's connection. A migration waiting for a lock gives
// up after three seconds, so the queries queued behind it on its table proceed, and the runner tries
// it again later.
var PostgresPresetMigrations = NewPostgresPreset(
	postgresConnection, env.PostgresMaxOpenConns, env.PostgresMaxIdleConns,
	pgdriver.WithConnParams(map[string]any{"lock_timeout": "3s"}),
)

// NewPostgresPreset returns a PostgreSQL preset whose pool is bounded before it opens. Extra options
// apply last: connection parameters replace any a DSN carries.
//
// Setting the limits on the handle afterwards stops working once anything has
// taken a connection, because the handle is cached; past that point they apply to
// nothing and report nothing.
func NewPostgresPreset(
	connection PostgresConnection,
	maxOpenConns int,
	maxIdleConns int,
	extra ...pgdriver.Option,
) *postgrespresets.Default {
	options := append(connection.options(), pgdriver.WithDialTimeout(postgresDialTimeout))
	preset := postgrespresets.NewDefault(append(options, extra...)...)
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
