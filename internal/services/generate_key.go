package services

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/a-novel-kit/context"
	"github.com/a-novel-kit/jwt/jwa"
	"github.com/a-novel-kit/jwt/jwk"

	"github.com/a-novel/authentication/config"
	"github.com/a-novel/authentication/internal/dao"
	"github.com/a-novel/authentication/internal/lib"
	"github.com/a-novel/authentication/models"
)

var (
	ErrED25519KeyGenerator = errors.New("ED25519KeyGenerator")
	ErrGenerateKeyService  = errors.New("GenerateKeyService.GenerateKey")
)

func NewErrED25519KeyGenerator(err error) error {
	return errors.Join(err, ErrED25519KeyGenerator)
}

func NewErrGenerateKeyService(err error) error {
	return errors.Join(err, ErrGenerateKeyService)
}

// KeyGenerator generates a new JSON Web Key private/public pair. It is a key-type agnostic wrapper around the
// JWT library generators.
type KeyGenerator func() (privateKey, publicKey *jwa.JWK, err error)

func ED25519KeyGenerator() (*jwa.JWK, *jwa.JWK, error) {
	private, public, err := jwk.GenerateED25519()
	if err != nil {
		return nil, nil, NewErrED25519KeyGenerator(fmt.Errorf("generate keypair: %w", err))
	}

	return private.JWK, public.JWK, nil
}

// GenerateKeyConfig provides information for generating a key for a target usage.
type GenerateKeyConfig struct {
	Generator KeyGenerator
	Config    config.Key
}

var generateKeysConfig = map[models.KeyUsage]GenerateKeyConfig{
	models.KeyUsageAuth: {
		Generator: ED25519KeyGenerator,
		Config:    config.Keys.Usages[models.KeyUsageAuth],
	},
	models.KeyUsageRefresh: {
		Generator: ED25519KeyGenerator,
		Config:    config.Keys.Usages[models.KeyUsageRefresh],
	},
}

type GenerateKeySource interface {
	SearchKeys(ctx context.Context, usage models.KeyUsage) ([]*dao.KeyEntity, error)
	InsertKey(ctx context.Context, data dao.InsertKeyData) (*dao.KeyEntity, error)
}

type GenerateKeyService struct {
	source GenerateKeySource
}

// GenerateKey generates a new key pair for a given usage. It uses the generateKeysConfig to generate the
// correct payload. Private key is encrypted using the master key before being saved in the database.
func (service *GenerateKeyService) GenerateKey(ctx context.Context, usage models.KeyUsage) (*uuid.UUID, error) {
	// Check the time last key was inserted for this usage, and compare to config. If last key is too recent,
	// return without generating a new key.
	keys, err := service.source.SearchKeys(ctx, usage)
	if err != nil {
		return nil, NewErrGenerateKeyService(fmt.Errorf("list keys: %w", err))
	}

	var lastCreated time.Time
	if len(keys) > 0 {
		lastCreated = keys[0].CreatedAt
	}

	// Last key was created within the rotation interval. No need to generate a new key.
	if time.Since(lastCreated) < generateKeysConfig[usage].Config.Rotation {
		return &keys[0].ID, nil
	}

	// Generate a new key pair.
	privateKey, publicKey, err := generateKeysConfig[usage].Generator()
	if err != nil {
		return nil, NewErrGenerateKeyService(fmt.Errorf("generate key pair: %w", err))
	}

	// Encrypt the private key using the master key, so it is protected against database dumping.
	privateKeyEncrypted, err := lib.EncryptMasterKey(ctx, privateKey)
	if err != nil {
		return nil, NewErrGenerateKeyService(fmt.Errorf("encrypt private key: %w", err))
	}

	// Encode values to base64 before saving them.
	privateKeyEncoded := base64.RawURLEncoding.EncodeToString(privateKeyEncrypted)

	// Extract the KID from the private key. Both public and private key should share the same KID.
	kid, err := uuid.Parse(privateKey.KID)
	if err != nil {
		return nil, NewErrGenerateKeyService(fmt.Errorf("parse KID: %w", err))
	}

	var publicKeyEncoded *string

	if publicKey != nil {
		// Serialize the public key.
		publicKeySerialized, err := json.Marshal(publicKey)
		if err != nil {
			return nil, NewErrGenerateKeyService(fmt.Errorf("serialize public key: %w", err))
		}

		publicKeyEncoded = lo.ToPtr(base64.RawURLEncoding.EncodeToString(publicKeySerialized))
	}

	// Insert the new key in the database.
	insertData := dao.InsertKeyData{
		ID:         kid,
		PrivateKey: privateKeyEncoded,
		PublicKey:  publicKeyEncoded,
		Usage:      usage,
		Now:        time.Now(),
		Expiration: time.Now().Add(generateKeysConfig[usage].Config.TTL),
	}

	if _, err = service.source.InsertKey(ctx, insertData); err != nil {
		return nil, NewErrGenerateKeyService(fmt.Errorf("insert key: %w", err))
	}

	return &kid, nil
}

func NewGenerateKeySource(searchDAO *dao.SearchKeysRepository, insertDAO *dao.InsertKeyRepository) GenerateKeySource {
	return &struct {
		dao.SearchKeysRepository
		dao.InsertKeyRepository
	}{
		SearchKeysRepository: *searchDAO,
		InsertKeyRepository:  *insertDAO,
	}
}

func NewGenerateKeyService(source GenerateKeySource) *GenerateKeyService {
	return &GenerateKeyService{source: source}
}
