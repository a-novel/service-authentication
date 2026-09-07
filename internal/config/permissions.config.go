package config

import "github.com/a-novel/service-authentication/v2/internal/config/auth"

// ErrUnknownRole reports a role absent from the permission configuration.
var ErrUnknownRole = auth.ErrUnknownRole

// Role bundles permissions and inheritance for a named role.
type Role = auth.Role

// Permissions maps role identifiers to their permission definitions.
type Permissions = auth.Permissions

const (
	// RoleAnon is the unauthenticated caller.
	RoleAnon = auth.RoleAnon
	// RoleUser is an authenticated standard user.
	RoleUser = auth.RoleUser
	// RoleAdmin is an operator with elevated access.
	RoleAdmin = auth.RoleAdmin
	// RoleSuperAdmin holds the highest level of access.
	RoleSuperAdmin = auth.RoleSuperAdmin
)
