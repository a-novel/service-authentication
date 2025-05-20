package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/service-authentication/api/codegen"
	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/services"
)

type SelectKeyService interface {
	SelectKey(ctx context.Context, request services.SelectKeyRequest) (*jwa.JWK, error)
}

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
