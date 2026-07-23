package config

import (
	_ "embed"
	"errors"
	"fmt"
)

// ErrUnknownRole reports a role name that the permission configuration does not
// define. A stored role that resolves to no entry is not "a role that grants
// nothing" — it is a role the application cannot interpret, and treating the two
// alike denied affected accounts everything with no signal.
var ErrUnknownRole = errors.New("unknown role")

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

// Priority returns the rank of a role, and ErrUnknownRole naming it when the
// configuration defines no such role. Callers ranking a role read from the
// database must use this rather than indexing Roles directly: a map miss yields
// the zero Role, priority 0 — the same rank as auth:anon — so an unknown role
// would silently outrank nobody and be outranked by everybody.
func (p Permissions) Priority(role string) (int, error) {
	r, ok := p.Roles[role]
	if !ok {
		return 0, fmt.Errorf("%w: %q", ErrUnknownRole, role)
	}

	return r.Priority, nil
}
