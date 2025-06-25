package config

import "os"

var DSN = os.Getenv("DSN")
