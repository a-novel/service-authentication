package config

import (
	_ "embed"

	"github.com/goccy/go-yaml"

	"github.com/a-novel-kit/golib/config"
)

//go:embed permissions.config.yaml
var defaultPermissionsFile []byte

var PermissionsConfigDefault = config.MustUnmarshal[Permissions](yaml.Unmarshal, defaultPermissionsFile)
