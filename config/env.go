package config

import "os"

var LoggerColor = os.Getenv("LOGGER_COLOR") == "true"
