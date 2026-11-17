package configtest

import (
	"os"

	loggingpresets "github.com/a-novel-kit/golib/logging/presets"
)

// Logger writes the logs of the code under test to standard output, as a local stack does.
var Logger = &loggingpresets.LogLocal{Out: os.Stdout}
