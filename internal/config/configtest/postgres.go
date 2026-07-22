// Package configtest holds shared test fixtures for the config package. Only `_test.go`
// files may import it, in this or any other package.
//
// The isolation is a convention the language does not enforce: Go links any package
// reachable from a production import into the binary, so keeping configtest off every
// production import path is what confines these fixtures to the test binary. A dedicated
// subpackage also sidesteps a naming trap. Go's build-time exclusion matches `_test.go`
// with an underscore. A `*.test.go` file ships in the binary.
package configtest

import (
	"github.com/a-novel/service-authentication/v2/internal/config"
)

// PostgresPreset is the PostgreSQL configuration used in integration tests. It aliases
// config.PostgresPresetDefault, so tests track the production preset with no parallel
// definition to maintain.
var PostgresPreset = config.PostgresPresetDefault
