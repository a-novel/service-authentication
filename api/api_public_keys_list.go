package api

import (
	"context"
	"fmt"
	"github.com/getsentry/sentry-go"

	"github.com/samber/lo"

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
	span := sentry.StartSpan(ctx, "API.ListPublicKeys")
	defer span.Finish()

	span.SetData("request.usage", params.Usage)

	keys, err := api.SearchKeysService.SearchKeys(span.Context(), services.SearchKeysRequest{
		Usage:   models.KeyUsage(params.Usage),
		Private: false,
	})
	if err != nil {
		span.SetData("service.err", err.Error())

		return nil, fmt.Errorf("search keys: %w", err)
	}

	keysModels, err := api.jwksToModels(keys...)
	if err != nil {
		span.SetData("conversion.err", err.Error())

		return nil, fmt.Errorf("convert keys to models: %w", err)
	}

	span.SetData("service.keys", keysModels)

	return lo.ToPtr(codegen.ListPublicKeysOKApplicationJSON(keysModels)), nil
}
