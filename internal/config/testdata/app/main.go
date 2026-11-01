// This program exposes the server's read timeout and strict rate limit presets, and the credentials
// of its JSON Keys connection, for subprocess tests.
package main

import (
	"fmt"

	"github.com/a-novel/service-authentication/v2/internal/config"
)

func main() {
	cfg := config.AppPresetDefault

	credentials := cfg.DependenciesConfig.ServiceJsonKeysCredentials

	fmt.Printf("%v %v %T %+v\n", cfg.Rest.Timeouts.Read, cfg.Limits.Strict, credentials, credentials)
}
