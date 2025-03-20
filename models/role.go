package models

// Role is a special attributes that is assigned to a session. It grants said session a given set of
// Permission.
type Role string

const (
	// RoleAnon represents an anonymous user.
	RoleAnon Role = "auth:anon"
	// RoleUser represents a regular user.
	RoleUser Role = "auth:user"
	// RoleAdmin represents an administrator.
	RoleAdmin Role = "auth:admin"
	// RoleSuperAdmin represents a super administrator.
	RoleSuperAdmin Role = "auth:super_admin"
)

var KnownRoles = []Role{
	RoleAnon,
	RoleUser,
}

type CredentialsRole string

const (
	CredentialsRoleUser       CredentialsRole = "user"
	CredentialsRoleAdmin      CredentialsRole = "admin"
	CredentialsRoleSuperAdmin CredentialsRole = "super_admin"
)

type CredentialRoleImportance int

const (
	CredentialRoleImportanceUnknown    CredentialRoleImportance = 0
	CredentialRoleImportanceUser       CredentialRoleImportance = 1
	CredentialRoleImportanceAdmin      CredentialRoleImportance = 2
	CredentialRoleImportanceSuperAdmin CredentialRoleImportance = 3
)

var KnownCredentialsRoles = []CredentialsRole{
	CredentialsRoleUser,
	CredentialsRoleAdmin,
	CredentialsRoleSuperAdmin,
}

var KnownCredentialsRolesWithImportance = map[CredentialsRole]CredentialRoleImportance{
	CredentialsRoleUser:       CredentialRoleImportanceUser,
	CredentialsRoleAdmin:      CredentialRoleImportanceAdmin,
	CredentialsRoleSuperAdmin: CredentialRoleImportanceSuperAdmin,
}
