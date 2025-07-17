package models

import (
	_ "embed"

	"github.com/goccy/go-yaml"

	"github.com/a-novel/golib/config"
)

// RoleConfig manages a set of permissions for a given Role.
type RoleConfig struct {
	// Inherits the permissions from every listed role. Circular dependencies between roles are not allowed.
	Inherits []Role `json:"inherits" yaml:"inherits"`
	// The set of permissions for the current role.
	Permissions []Permission `json:"permissions" yaml:"permissions"`
}

// PermissionsConfig maps every Role to a set of permissions.
type PermissionsConfig struct {
	Roles map[Role]RoleConfig `json:"roles" yaml:"roles"`
}

//go:embed permissions.yaml
var defaultPermissionsFile []byte

var DefaultPermissionsConfig = config.MustUnmarshal[PermissionsConfig](yaml.Unmarshal, defaultPermissionsFile)
