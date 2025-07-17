package api

import (
	"github.com/samber/lo"

	"github.com/a-novel/service-authentication/models"
	"github.com/a-novel/service-authentication/models/api"
)

func (api *API) CredentialsRoleToModel(role apimodels.CredentialsRole) models.CredentialsRole {
	return lo.Switch[apimodels.CredentialsRole, models.CredentialsRole](role).
		Case(apimodels.CredentialsRoleUser, models.CredentialsRoleUser).
		Case(apimodels.CredentialsRoleAdmin, models.CredentialsRoleAdmin).
		Case(apimodels.CredentialsRoleSuperAdmin, models.CredentialsRoleSuperAdmin).
		Default("")
}

func (api *API) CredentialsRoleFromModel(role models.CredentialsRole) apimodels.CredentialsRole {
	return lo.Switch[models.CredentialsRole, apimodels.CredentialsRole](role).
		Case(models.CredentialsRoleUser, apimodels.CredentialsRoleUser).
		Case(models.CredentialsRoleAdmin, apimodels.CredentialsRoleAdmin).
		Case(models.CredentialsRoleSuperAdmin, apimodels.CredentialsRoleSuperAdmin).
		Default("")
}
