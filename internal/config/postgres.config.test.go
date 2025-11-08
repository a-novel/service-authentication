package config

import (
	"github.com/uptrace/bun/driver/pgdriver"

	postgrespresets "github.com/a-novel/golib/postgres/presets"

	"github.com/a-novel/service-authentication/internal/config/env"
)

var PostgresPresetTest = postgrespresets.NewDefault(pgdriver.WithDSN(env.PostgresDsnTest))
