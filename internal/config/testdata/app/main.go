// This program exposes the server's read timeout and strict rate limit presets for subprocess tests.
package main

import (
	"fmt"

	"github.com/a-novel/service-authentication/v2/internal/config"
)

func main() {
	fmt.Println(config.AppPresetDefault.Rest.Timeouts.Read, config.AppPresetDefault.Limits.Strict)
}
