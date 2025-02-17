package models

// RoleConfig manages a set of permissions for a given Role.
type RoleConfig struct {
	// Inherits the permissions from every listed role. Circular dependencies between roles are not allowed.
	Inherits []Role `yaml:"inherits"`
	// The set of permissions for the current role.
	Permissions []Permission `yaml:"permissions"`
}

// PermissionsConfig maps every Role to a set of permissions.
type PermissionsConfig struct {
	Roles map[Role]RoleConfig `yaml:"roles"`
}
