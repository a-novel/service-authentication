// Package configtest holds the configuration fixtures tests share across packages. Only
// `_test.go` files import it, which keeps its fixtures out of production binaries.
package configtest

import (
	"github.com/a-novel/service-authentication/v2/internal/config"
)

// PostgresPreset is the PostgreSQL configuration of integration tests: the production preset.
var PostgresPreset = config.PostgresPresetDefault
