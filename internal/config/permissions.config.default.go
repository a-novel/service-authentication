package config

import (
	_ "embed"

	"github.com/goccy/go-yaml"

	"github.com/a-novel-kit/golib/config"
)

//go:embed permissions.config.yaml
var defaultPermissionsFile []byte

// PermissionsConfigDefault is the built-in role/permission map, loaded from the
// embedded permissions.config.yaml. It applies when no custom permission set is
// supplied.
var PermissionsConfigDefault = config.MustUnmarshal[Permissions](yaml.Unmarshal, defaultPermissionsFile)
