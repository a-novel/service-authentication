package config

import (
	_ "embed"

	"github.com/a-novel/service-authentication/models"
)

// Role manages a set of permissions for a given Role.
type Role struct {
	// Inherits the permissions from every listed role. Circular dependencies between roles are not allowed.
	Inherits []models.Role `json:"inherits" yaml:"inherits"`
	// The set of permissions for the current role.
	Permissions []models.Permission `json:"permissions" yaml:"permissions"`
}

// Permissions maps every Role to a set of permissions.
type Permissions struct {
	Roles map[models.Role]Role `json:"roles" yaml:"roles"`
}
