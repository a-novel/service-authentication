// Package configtest holds shared test fixtures for the config package. Production code must
// not import it; only test files (in this or any other package's `_test.go` files) may.
//
// Isolation is enforced by convention, not by the language. Go links any package reachable
// from a production import into the binary, regardless of where it lives — keeping configtest
// out of every production import path is what guarantees these fixtures ship only with the
// test binary. The split exists because the alternative — putting the fixtures next to the
// production config files in a `*.test.go` file — does ship them: Go's build-time exclusion
// rule only matches `_test.go` (with an underscore), not `.test.go` (with a dot). A dedicated
// subpackage makes the boundary explicit and easy to lint or grep against.
package configtest

import (
	"github.com/a-novel/service-authentication/v2/internal/config"
)

// PostgresPreset is the PostgreSQL configuration used in integration tests. It aliases
// config.PostgresPresetDefault so tests inherit any future change to the production preset
// (timeouts, retry policy, additional dial options) without a parallel update here.
var PostgresPreset = config.PostgresPresetDefault
