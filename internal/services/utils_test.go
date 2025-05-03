package services_test

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/context"
	"github.com/a-novel-kit/jwt/jwa"
	"github.com/a-novel-kit/jwt/jwk"

	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/internal/services"
)

func mustEncryptValue(ctx context.Context, t *testing.T, data any) []byte {
	t.Helper()

	res, err := lib.EncryptMasterKey(ctx, data)
	if err != nil {
		panic(err)
	}

	return res
}

func mustEncryptBase64Value(ctx context.Context, t *testing.T, data any) string {
	t.Helper()

	res := mustEncryptValue(ctx, t, data)

	return base64.RawURLEncoding.EncodeToString(res)
}

func mustSerializeBase64Value(t *testing.T, data any) string {
	t.Helper()

	res, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	return base64.RawURLEncoding.EncodeToString(res)
}

func generateAuthTokenKeySet(t *testing.T, size int) ([]*jwk.Key[ed25519.PrivateKey], []*jwk.Key[ed25519.PublicKey]) {
	t.Helper()

	privateKeys := make([]*jwk.Key[ed25519.PrivateKey], size)
	publicKeys := make([]*jwk.Key[ed25519.PublicKey], size)

	for i := range size {
		privateKey, publicKey, err := jwk.GenerateED25519()
		require.NoError(t, err)

		privateKeys[i] = privateKey
		publicKeys[i] = publicKey
	}

	return privateKeys, publicKeys
}

func mustIssueToken(
	t *testing.T, key *jwk.Key[ed25519.PrivateKey], request services.IssueTokenRequest,
) string {
	t.Helper()

	source := jwk.NewED25519PrivateSource(jwk.SourceConfig{
		Fetch: func(_ context.Context) ([]*jwa.JWK, error) {
			return []*jwa.JWK{key.JWK}, nil
		},
	})

	issuer := services.NewIssueTokenService(source)
	token, err := issuer.IssueToken(t.Context(), request)
	require.NoError(t, err)

	return token
}

func mustIssueRefreshToken(
	t *testing.T, key *jwk.Key[ed25519.PrivateKey], request services.IssueRefreshTokenRequest,
) string {
	t.Helper()

	source := jwk.NewED25519PrivateSource(jwk.SourceConfig{
		Fetch: func(_ context.Context) ([]*jwa.JWK, error) {
			return []*jwa.JWK{key.JWK}, nil
		},
	})

	issuer := services.NewIssueRefreshTokenService(source)
	token, err := issuer.IssueRefreshToken(t.Context(), request)
	require.NoError(t, err)

	return token
}
