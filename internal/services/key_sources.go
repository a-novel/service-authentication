package services

import (
	"context"

	"golang.org/x/crypto/ed25519"

	"github.com/a-novel-kit/jwt/jwa"
	"github.com/a-novel-kit/jwt/jwk"

	"github.com/a-novel/service-authentication/config"
	"github.com/a-novel/service-authentication/models"
)

// KeysProviderSource is the interface for jwk.Source providers to source keys from the database.
type KeysProviderSource interface {
	SearchKeys(ctx context.Context, request SearchKeysRequest) ([]*jwa.JWK, error)
}

func NewAuthPrivateKeysProvider(source KeysProviderSource) *jwk.Source[ed25519.PrivateKey] {
	return jwk.NewED25519PrivateSource(jwk.SourceConfig{
		CacheDuration: config.Keys.Source.Cache.Duration,
		Fetch: func(ctx context.Context) ([]*jwa.JWK, error) {
			return source.SearchKeys(ctx, SearchKeysRequest{Usage: models.KeyUsageAuth, Private: true})
		},
	})
}

func NewAuthPublicKeysProvider(source KeysProviderSource) *jwk.Source[ed25519.PublicKey] {
	return jwk.NewED25519PublicSource(jwk.SourceConfig{
		CacheDuration: config.Keys.Source.Cache.Duration,
		Fetch: func(ctx context.Context) ([]*jwa.JWK, error) {
			return source.SearchKeys(ctx, SearchKeysRequest{Usage: models.KeyUsageAuth})
		},
	})
}

func NewRefreshPrivateKeysProvider(source KeysProviderSource) *jwk.Source[ed25519.PrivateKey] {
	return jwk.NewED25519PrivateSource(jwk.SourceConfig{
		CacheDuration: config.Keys.Source.Cache.Duration,
		Fetch: func(ctx context.Context) ([]*jwa.JWK, error) {
			return source.SearchKeys(ctx, SearchKeysRequest{Usage: models.KeyUsageRefresh, Private: true})
		},
	})
}

func NewRefreshPublicKeysProvider(source KeysProviderSource) *jwk.Source[ed25519.PublicKey] {
	return jwk.NewED25519PublicSource(jwk.SourceConfig{
		CacheDuration: config.Keys.Source.Cache.Duration,
		Fetch: func(ctx context.Context) ([]*jwa.JWK, error) {
			return source.SearchKeys(ctx, SearchKeysRequest{Usage: models.KeyUsageRefresh})
		},
	})
}
