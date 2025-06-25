package services

import (
	"context"
	"errors"
	"fmt"
	"github.com/getsentry/sentry-go"

	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/models"
)

var ErrSearchKeysService = errors.New("SearchKeysService.SearchKeys")

func NewErrSearchKeysService(err error) error {
	return errors.Join(err, ErrSearchKeysService)
}

// SearchKeysSource is the source used to perform the SearchKeysService.SearchKeys action.
type SearchKeysSource interface {
	SearchKeys(ctx context.Context, usage models.KeyUsage) ([]*dao.KeyEntity, error)
}

// SearchKeysRequest is the input used to perform the SearchKeysService.SearchKeys action.
type SearchKeysRequest struct {
	// Usage expected for the keys.
	Usage models.KeyUsage
	// If true, returns the private key. Otherwise, return the public key.
	Private bool
}

// SearchKeysService is the service used to perform the SearchKeysService.SearchKeys action.
//
// You may create one using the NewSearchKeysService function.
type SearchKeysService struct {
	source SearchKeysSource
}

// SearchKeys retrieves a batch of keys from the source. All keys are serialized, and match the usage required
// by the request.
func (service *SearchKeysService) SearchKeys(ctx context.Context, request SearchKeysRequest) ([]*jwa.JWK, error) {
	span := sentry.StartSpan(ctx, "SearchKeysService.SearchKeys")
	defer span.Finish()

	span.SetData("request.usage", request.Usage)
	span.SetData("request.private", request.Private)

	keys, err := service.source.SearchKeys(span.Context(), request.Usage)
	if err != nil {
		span.SetData("dao.error", err.Error())

		return nil, NewErrSearchKeysService(fmt.Errorf("search keys: %w", err))
	}

	span.SetData("keys.count", len(keys))

	deserialized := make([]*jwa.JWK, len(keys))

	for i, key := range keys {
		subSpan := sentry.StartSpan(span.Context(), "deserializeKey")
		subSpan.SetData("key.id", key.ID)

		deserialized[i], err = dao.ConsumeKey(subSpan.Context(), key, request.Private)
		if err != nil {
			subSpan.SetData("deserializeKey.error", err.Error())
			subSpan.Finish()

			return nil, NewErrSearchKeysService(fmt.Errorf("consume DAO key (kid %s): %w", key.ID, err))
		}

		subSpan.Finish()
	}

	return deserialized, nil
}

func NewSearchKeysService(source SearchKeysSource) *SearchKeysService {
	return &SearchKeysService{source: source}
}
