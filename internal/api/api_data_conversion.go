package api

import (
	"github.com/samber/lo"

	"github.com/a-novel/service-authentication/internal/api/codegen"
	"github.com/a-novel/service-authentication/models"
)

func (api *API) CredentialsRoleToModel(role codegen.CredentialsRole) models.CredentialsRole {
	return lo.Switch[codegen.CredentialsRole, models.CredentialsRole](role).
		Case(codegen.CredentialsRoleUser, models.CredentialsRoleUser).
		Case(codegen.CredentialsRoleAdmin, models.CredentialsRoleAdmin).
		Case(codegen.CredentialsRoleSuperAdmin, models.CredentialsRoleSuperAdmin).
		Default("")
}

func (api *API) CredentialsRoleFromModel(role models.CredentialsRole) codegen.CredentialsRole {
	return lo.Switch[models.CredentialsRole, codegen.CredentialsRole](role).
		Case(models.CredentialsRoleUser, codegen.CredentialsRoleUser).
		Case(models.CredentialsRoleAdmin, codegen.CredentialsRoleAdmin).
		Case(models.CredentialsRoleSuperAdmin, codegen.CredentialsRoleSuperAdmin).
		Default("")
}
