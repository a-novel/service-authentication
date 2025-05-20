package services_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ed25519"

	"github.com/a-novel-kit/jwt"
	"github.com/a-novel-kit/jwt/jwa"
	"github.com/a-novel-kit/jwt/jwk"
	"github.com/a-novel-kit/jwt/jws"

	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
)

func TestIssueToken(t *testing.T) {
	t.Parallel()

	privateKeys, publicKeys := generateAuthTokenKeySet(t, 1)

	privateKeysJSON := lo.Map(privateKeys, func(item *jwk.Key[ed25519.PrivateKey], _ int) *jwa.JWK {
		return item.JWK
	})

	testCases := []struct {
		name string

		publicKey *jwk.Key[ed25519.PublicKey]
		request   services.IssueTokenRequest

		expect models.AccessTokenClaims
	}{
		{
			name: "Success",

			publicKey: publicKeys[0],
			request: services.IssueTokenRequest{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				Roles:  []models.Role{models.RoleUser},
			},

			expect: models.AccessTokenClaims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				Roles:  []models.Role{models.RoleUser},
			},
		},
		{
			name: "AnonymousToken",

			publicKey: publicKeys[0],
			request: services.IssueTokenRequest{
				Roles: []models.Role{models.RoleAnon},
			},

			expect: models.AccessTokenClaims{
				Roles: []models.Role{models.RoleAnon},
			},
		},
		{
			name: "RefreshTokenID",

			publicKey: publicKeys[0],
			request: services.IssueTokenRequest{
				UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				Roles:          []models.Role{models.RoleUser},
				RefreshTokenID: lo.ToPtr("00000000-0000-0000-0000-000000000002"),
			},

			expect: models.AccessTokenClaims{
				UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				Roles:          []models.Role{models.RoleUser},
				RefreshTokenID: lo.ToPtr("00000000-0000-0000-0000-000000000002"),
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			source := jwk.NewED25519PrivateSource(jwk.SourceConfig{
				Fetch: func(_ context.Context) ([]*jwa.JWK, error) {
					return privateKeysJSON, nil
				},
			})

			service := services.NewIssueTokenService(source)

			data, err := service.IssueToken(t.Context(), testCase.request)
			require.NoError(t, err)

			t.Run("AccessToken", func(t *testing.T) {
				verifier := jws.NewED25519Verifier(testCase.publicKey.Key())
				recipient := jwt.NewRecipient(jwt.RecipientConfig{
					Plugins: []jwt.RecipientPlugin{verifier},
				})

				var claims models.AccessTokenClaims

				require.NoError(t, recipient.Consume(t.Context(), data, &claims))
				require.Equal(t, testCase.expect, claims)
			})
		})
	}
}
