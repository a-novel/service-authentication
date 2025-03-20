package api

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/authentication/api/codegen"
	"github.com/a-novel/authentication/models"
)

func (api *API) jwkToModel(src *jwa.JWK) (*codegen.JWK, error) {
	rawPayload := new(codegen.JWKAdditional)
	if err := rawPayload.UnmarshalJSON(src.Payload); err != nil {
		return nil, fmt.Errorf("unmarshal payload: %w", err)
	}

	kid, err := uuid.Parse(src.KID)
	if err != nil {
		return nil, fmt.Errorf("parse kid: %w", err)
	}

	return &codegen.JWK{
		Kty:             codegen.KTY(src.KTY),
		Use:             codegen.Use(src.Use),
		KeyOps:          lo.Map(src.KeyOps, func(item jwa.KeyOp, _ int) codegen.KeyOp { return codegen.KeyOp(item) }),
		Alg:             codegen.Alg(src.Alg),
		Kid:             codegen.OptKID{Value: codegen.KID(kid), Set: true},
		AdditionalProps: *rawPayload,
	}, nil
}

func (api *API) jwksToModels(src ...*jwa.JWK) ([]codegen.JWK, error) {
	output := make([]codegen.JWK, len(src))

	for i, jwk := range src {
		model, err := api.jwkToModel(jwk)
		if err != nil {
			return nil, fmt.Errorf("convert jwk to model: %w", err)
		}

		output[i] = *model
	}

	return output, nil
}

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
