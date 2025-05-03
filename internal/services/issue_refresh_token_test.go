package services_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ed25519"

	"github.com/a-novel-kit/context"
	"github.com/a-novel-kit/jwt"
	"github.com/a-novel-kit/jwt/jwa"
	"github.com/a-novel-kit/jwt/jwk"
	"github.com/a-novel-kit/jwt/jws"

	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
)

func TestIssueRefreshToken(t *testing.T) {
	t.Parallel()

	privateKeys, publicKeys := generateAuthTokenKeySet(t, 1)

	privateKeysJSON := lo.Map(privateKeys, func(item *jwk.Key[ed25519.PrivateKey], _ int) *jwa.JWK {
		return item.JWK
	})

	testCases := []struct {
		name string

		publicKey *jwk.Key[ed25519.PublicKey]
		request   services.IssueRefreshTokenRequest

		expect    models.RefreshTokenClaims
		expectErr error
	}{
		{
			name: "Success",

			publicKey: publicKeys[0],
			request: services.IssueRefreshTokenRequest{
				Claims: &models.AccessTokenClaims{
					UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:  []models.Role{models.RoleUser},
				},
			},

			expect: models.RefreshTokenClaims{
				UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		},
		{
			name: "AnonymousToken",

			publicKey: publicKeys[0],
			request: services.IssueRefreshTokenRequest{
				Claims: &models.AccessTokenClaims{
					Roles: []models.Role{models.RoleAnon},
				},
			},

			expectErr: services.ErrRefreshTokenWithAnonSession,
		},
		{
			name: "RefreshRefreshToken",

			publicKey: publicKeys[0],
			request: services.IssueRefreshTokenRequest{
				Claims: &models.AccessTokenClaims{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:          []models.Role{models.RoleUser},
					RefreshTokenID: lo.ToPtr("00000000-0000-0000-0000-000000000002"),
				},
			},

			expectErr: services.ErrRefreshRefreshToken,
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

			service := services.NewIssueRefreshTokenService(source)

			data, err := service.IssueRefreshToken(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)

			if err == nil {
				verifier := jws.NewED25519Verifier(testCase.publicKey.Key())
				recipient := jwt.NewRecipient(jwt.RecipientConfig{
					Plugins: []jwt.RecipientPlugin{verifier},
				})

				var claims models.RefreshTokenClaims

				require.NoError(t, recipient.Consume(t.Context(), data, &claims))
				require.Equal(t, testCase.expect, claims)
			}
		})
	}
}
