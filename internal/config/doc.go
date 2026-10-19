// Package config assembles the authentication service's configuration from the process
// environment, with a default for every unset variable.
//
// The auth subpackage holds the static settings the published Go client shares, env reads
// the environment, and configtest holds the fixtures tests share. Runtime code reads the
// typed [App] built here rather than calling os.Getenv.
package config
