package models

// Role is a special attributes that is assigned to a session. It grants said session a given set of
// Permission.
type Role string

const (
	// RoleAnon represents an anonymous user.
	RoleAnon Role = "auth:anon"
	// RoleUser represents a regular user.
	RoleUser Role = "auth:user"
)

var KnownRoles = []Role{
	RoleAnon,
	RoleUser,
}
