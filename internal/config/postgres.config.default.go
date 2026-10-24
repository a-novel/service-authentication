package config

import (
	"time"

	"github.com/samber/lo"
	"github.com/uptrace/bun/driver/pgdriver"

	postgrespresets "github.com/a-novel-kit/golib/postgres/presets"

	"github.com/a-novel/service-authentication/v2/internal/config/env"
)

// postgresDialTimeout covers the network a Cloud Run instance with Direct VPC egress provisions
// after it starts. The first connection can wait minutes for it.
const postgresDialTimeout = 3 * time.Minute

// postgresConnection reaches PostgreSQL through the POSTGRES_* variables.
var postgresConnection = postgrespresets.Connection{
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

// NewPostgresPreset returns a PostgreSQL preset with a three-minute dial timeout and a pool bounded
// before it opens. Extra options apply last. It panics when the connection names a host without a
// password.
//
// The pool limits are set before the handle is first used: once the handle is cached, later changes
// to them are silently ignored.
func NewPostgresPreset(
	connection postgrespresets.Connection,
	maxOpenConns int,
	maxIdleConns int,
	extra ...pgdriver.Option,
) *postgrespresets.Default {
	connection.DialTimeout = postgresDialTimeout
	options := lo.Must(connection.Options())

	preset := postgrespresets.NewDefault(append(options, extra...)...)
	preset.MaxOpenConns = maxOpenConns
	preset.MaxIdleConns = maxIdleConns

	return preset
}
