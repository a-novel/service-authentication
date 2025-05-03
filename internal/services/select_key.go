package services

import (
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel-kit/context"
	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/lib"
)

var ErrSelectKeyService = errors.New("SelectKeyService.SelectKey")

func NewErrSelectKeyService(err error) error {
	return errors.Join(err, ErrSelectKeyService)
}

// SelectKeySource is the source used to perform the SelectKeyService.SelectKey action.
type SelectKeySource interface {
	SelectKey(ctx context.Context, id uuid.UUID) (*dao.KeyEntity, error)
}

// SelectKeyRequest is the input used to perform the SelectKeyService.SelectKey action.
type SelectKeyRequest struct {
	// ID of the target key. This matches the "kid" header.
	ID uuid.UUID
	// If true, return the private key. Otherwise, return the public key.
	Private bool
}

// SelectKeyService is the service used to perform the SelectKeyService.SelectKey action.
//
// You may create one using the NewSelectKeyService function.
type SelectKeyService struct {
	source SelectKeySource
}

func (service *SelectKeyService) SelectKey(ctx context.Context, request SelectKeyRequest) (*jwa.JWK, error) {
	key, err := service.source.SelectKey(ctx, request.ID)
	if err != nil {
		return nil, NewErrSelectKeyService(fmt.Errorf("select key: %w", err))
	}

	deserialized, err := lib.ConsumeDAOKey(ctx, key, request.Private)
	if err != nil {
		return nil, NewErrSelectKeyService(fmt.Errorf("consume DAO key (kid %s): %w", key.ID, err))
	}

	return deserialized, nil
}

func NewSelectKeyService(source SelectKeySource) *SelectKeyService {
	return &SelectKeyService{source: source}
}
