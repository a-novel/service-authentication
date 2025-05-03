package api

import (
	"fmt"

	"github.com/samber/lo"

	"github.com/a-novel-kit/context"
	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/service-authentication/api/codegen"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
)

type SearchKeysService interface {
	SearchKeys(ctx context.Context, request services.SearchKeysRequest) ([]*jwa.JWK, error)
}

func (api *API) ListPublicKeys(
	ctx context.Context, params codegen.ListPublicKeysParams,
) (codegen.ListPublicKeysRes, error) {
	keys, err := api.SearchKeysService.SearchKeys(ctx, services.SearchKeysRequest{
		Usage:   models.KeyUsage(params.Usage),
		Private: false,
	})
	if err != nil {
		return nil, fmt.Errorf("search keys: %w", err)
	}

	keysModels, err := api.jwksToModels(keys...)
	if err != nil {
		return nil, fmt.Errorf("convert keys to models: %w", err)
	}

	return lo.ToPtr(codegen.ListPublicKeysOKApplicationJSON(keysModels)), nil
}
