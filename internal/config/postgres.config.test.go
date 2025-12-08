package config

import (
	"github.com/uptrace/bun/driver/pgdriver"

	postgrespresets "github.com/a-novel-kit/golib/postgres/presets"

	"github.com/a-novel/service-authentication/v2/internal/config/env"
)

var PostgresPresetTest = postgrespresets.NewDefault(pgdriver.WithDSN(env.PostgresDsnTest))
