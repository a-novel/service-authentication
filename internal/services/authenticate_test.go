package services_test

import (
	"crypto/ed25519"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/context"
	"github.com/a-novel-kit/jwt/jwa"
	"github.com/a-novel-kit/jwt/jwk"

	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
)

func TestAuthenticate(t *testing.T) {
	t.Parallel()

	privateKeys, publicKeys := generateAuthTokenKeySet(t, 3)

	publicKeysJSON := lo.Map(publicKeys, func(item *jwk.Key[ed25519.PublicKey], _ int) *jwa.JWK {
		return item.JWK
	})

	fakePrivateKey, _, err := jwk.GenerateED25519()
	require.NoError(t, err)

	testCases := []struct {
		name string

		request string

		expect    *models.AccessTokenClaims
		expectErr error
	}{
		{
			name: "Success",

			request: mustIssueToken(t, privateKeys[0], services.IssueTokenRequest{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				Roles:  []models.Role{"foo", "bar"},
			}),

			expect: &models.AccessTokenClaims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				Roles:  []models.Role{"foo", "bar"},
			},
		},
		{
			name: "Success/OlderKey",

			request: mustIssueToken(t, privateKeys[1], services.IssueTokenRequest{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				Roles:  []models.Role{"foo", "bar"},
			}),

			expect: &models.AccessTokenClaims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				Roles:  []models.Role{"foo", "bar"},
			},
		},
		{
			name: "Success/Anonymous",

			request: mustIssueToken(t, privateKeys[0], services.IssueTokenRequest{
				Roles: []models.Role{"foo", "bar"},
			}),

			expect: &models.AccessTokenClaims{
				Roles: []models.Role{"foo", "bar"},
			},
		},
		{
			name: "Error/InvalidSignature",

			request: mustIssueToken(t, fakePrivateKey, services.IssueTokenRequest{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				Roles:  []models.Role{"foo", "bar"},
			}),

			expectErr: models.ErrUnauthorized,
		},
		{
			name: "Error/Unsigned",

			request: "",

			expectErr: models.ErrUnauthorized,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			source := jwk.NewED25519PublicSource(jwk.SourceConfig{
				Fetch: func(_ context.Context) ([]*jwa.JWK, error) {
					return publicKeysJSON, nil
				},
			})

			service := services.NewAuthenticateService(source)

			data, err := service.Authenticate(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, data)
		})
	}
}
