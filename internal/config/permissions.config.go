package config

import (
	_ "embed"
)

// A Role bundles the permissions granted to its holders. A role may inherit other
// roles to accumulate their permissions, and its priority ranks it against the rest
// of the hierarchy.
type Role struct {
	// Inherits pulls in the permissions of every listed role. Circular inheritance
	// between roles is not allowed.
	Inherits    []string `json:"inherits"    yaml:"inherits"`
	Permissions []string `json:"permissions" yaml:"permissions"`
	// Priority ranks this role in the hierarchy; a higher value outranks a lower one.
	Priority int `json:"priority" yaml:"priority"`
}

// Built-in role identifiers. Each matches a key in the permissions map, and they
// grant progressively more access.
const (
	// RoleAnon is the unauthenticated caller.
	RoleAnon = "auth:anon"
	// RoleUser is an authenticated standard user.
	RoleUser = "auth:user"
	// RoleAdmin is an operator with elevated access.
	RoleAdmin = "auth:admin"
	// RoleSuperAdmin holds the highest level of access.
	RoleSuperAdmin = "auth:superadmin"
)

// Permissions is the role/permission configuration: it maps each role identifier
// to its Role definition.
type Permissions struct {
	Roles map[string]Role `json:"roles" yaml:"roles"`
}
