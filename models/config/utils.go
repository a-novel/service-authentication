package config

import "os"

// EnvPrefix allows to set a custom prefix to all configuration environment variables.
// This is useful when importing the package in another project, when env variable names
// might conflict with the source project.
var EnvPrefix = os.Getenv("SERVICE_AUTHENTICATION_ENV")

func getEnv(name string) string {
	if EnvPrefix != "" {
		return os.Getenv(EnvPrefix + "_" + name)
	}

	return os.Getenv(name)
}
