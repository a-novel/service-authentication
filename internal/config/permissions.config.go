package config

import (
	_ "embed"
)

// Role manages a set of permissions for a given Role.
type Role struct {
	// Inherits the permissions from every listed role. Circular dependencies between roles are not allowed.
	Inherits []string `json:"inherits" yaml:"inherits"`
	// The set of permissions for the current role.
	Permissions []string `json:"permissions" yaml:"permissions"`
	// The hierarchy level of this role. Roles with higher priorities are leaders, and the ones with lower
	// priorities are subordinates.
	Priority int `json:"priority" yaml:"priority"`
}

const (
	RoleAnon       = "auth:anon"
	RoleUser       = "auth:user"
	RoleAdmin      = "auth:admin"
	RoleSuperAdmin = "auth:superadmin"
)

// Permissions maps every Role to a set of permissions.
type Permissions struct {
	Roles map[string]Role `json:"roles" yaml:"roles"`
}
