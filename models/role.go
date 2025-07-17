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

type CredentialsRole string

func (role CredentialsRole) String() string {
	return string(role)
}

type CredentialsRoles []CredentialsRole

func (roles CredentialsRoles) Strings() []string {
	strs := make([]string, len(roles))
	for i, role := range roles {
		strs[i] = role.String()
	}

	return strs
}

const (
	CredentialsRoleUser       CredentialsRole = "user"
	CredentialsRoleAdmin      CredentialsRole = "admin"
	CredentialsRoleSuperAdmin CredentialsRole = "super_admin"
)

type CredentialRoleImportance int

func (importance CredentialRoleImportance) Int() int {
	return int(importance)
}

const (
	CredentialRoleImportanceUnknown    CredentialRoleImportance = 0
	CredentialRoleImportanceUser       CredentialRoleImportance = 1
	CredentialRoleImportanceAdmin      CredentialRoleImportance = 2
	CredentialRoleImportanceSuperAdmin CredentialRoleImportance = 3
)

var KnownCredentialsRolesWithImportance = map[CredentialsRole]CredentialRoleImportance{
	CredentialsRoleUser:       CredentialRoleImportanceUser,
	CredentialsRoleAdmin:      CredentialRoleImportanceAdmin,
	CredentialsRoleSuperAdmin: CredentialRoleImportanceSuperAdmin,
}
