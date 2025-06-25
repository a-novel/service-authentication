package api

import (
	"context"
	"errors"
	"fmt"
	"github.com/getsentry/sentry-go"

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
	span := sentry.StartSpan(ctx, "API.GetPublicKey")
	defer span.Finish()

	span.SetData("request.kid", params.Kid)

	key, err := api.SelectKeyService.SelectKey(span.Context(), services.SelectKeyRequest{
		ID:      uuid.UUID(params.Kid),
		Private: false,
	})

	switch {
	case errors.Is(err, dao.ErrKeyNotFound):
		span.SetData("service.err", err.Error())

		return &codegen.NotFoundError{Error: "key not found"}, nil
	case err != nil:
		span.SetData("service.err", err.Error())

		return nil, fmt.Errorf("select key: %w", err)
	}

	span.SetData("service.key", key.JWKCommon)

	return api.jwkToModel(key)
}
