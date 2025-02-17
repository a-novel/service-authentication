package api

import (
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel-kit/context"

	"github.com/a-novel/authentication/api/codegen"
	"github.com/a-novel/authentication/internal/dao"
	"github.com/a-novel/authentication/internal/services"
)

func (api *API) GetPublicKey(ctx context.Context, params codegen.GetPublicKeyParams) (codegen.GetPublicKeyRes, error) {
	key, err := api.SelectKeyService.SelectKey(ctx, services.SelectKeyRequest{
		ID:      uuid.UUID(params.Kid),
		Private: false,
	})

	switch {
	case errors.Is(err, dao.ErrKeyNotFound):
		return &codegen.NotFoundError{Error: "key not found"}, nil
	case err != nil:
		return nil, fmt.Errorf("select key: %w", err)
	}

	return api.jwkToModel(key)
}
