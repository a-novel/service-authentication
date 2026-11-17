package auth

import (
	"errors"
	"fmt"
	"slices"
)

var (
	// ErrUnknownRole reports a role name the permission configuration does not define.
	ErrUnknownRole = errors.New("unknown role")
	// ErrCircularInheritance reports roles that inherit from each other.
	ErrCircularInheritance = errors.New("circular role inheritance")
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

// Priority returns the rank of a role, and ErrUnknownRole naming it when the
// configuration defines no such role. Rank a role read from the database with it:
// indexing Roles directly ranks an unknown role 0, as auth:anon.
func (p Permissions) Priority(role string) (int, error) {
	r, ok := p.Roles[role]
	if !ok {
		return 0, fmt.Errorf("%w: %q", ErrUnknownRole, role)
	}

	return r.Priority, nil
}

// ByRole maps each role to every permission it grants: its own, then those of each role it
// inherits, transitively. It returns [ErrUnknownRole] when a role inherits one the configuration
// does not define, and [ErrCircularInheritance] when roles inherit from each other.
func (p Permissions) ByRole() (map[string][]string, error) {
	byRole := make(map[string][]string, len(p.Roles))
	resolving := map[string]bool{}

	for name := range p.Roles {
		_, err := p.resolve(name, byRole, resolving)
		if err != nil {
			return nil, err
		}
	}

	return byRole, nil
}

// resolve returns the permissions the named role grants, and stores them in byRole. resolving holds
// the roles whose resolution is under way, so meeting one of them again closes a cycle. The
// recursion follows the configured inheritance, whose depth stays small.
func (p Permissions) resolve(name string, byRole map[string][]string, resolving map[string]bool) ([]string, error) {
	if permissions, resolved := byRole[name]; resolved {
		return permissions, nil
	}

	role, ok := p.Roles[name]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownRole, name)
	}

	if resolving[name] {
		return nil, fmt.Errorf("%w: %q", ErrCircularInheritance, name)
	}

	resolving[name] = true

	permissions := slices.Clone(role.Permissions)

	for _, parent := range role.Inherits {
		inherited, err := p.resolve(parent, byRole, resolving)
		if err != nil {
			return nil, err
		}

		permissions = append(permissions, inherited...)
	}

	byRole[name] = permissions

	return permissions, nil
}
