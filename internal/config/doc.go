// Package config holds the deployment-time configuration for the authentication
// service. It defines the typed shape of the application config (REST server,
// Postgres, SMTP, role/permission map, short-code lifetimes, language settings,
// observability) and the defaults applied when an environment variable is unset.
//
// The env subpackage parses the process environment into the [App] struct;
// configtest exposes a fixture builder for tests. Runtime code should depend on
// the typed structs declared here rather than reading os.Getenv directly.
package config
