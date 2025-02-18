package services_test

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ed25519"

	"github.com/a-novel-kit/context"
	"github.com/a-novel-kit/jwt"
	"github.com/a-novel-kit/jwt/jwa"
	"github.com/a-novel-kit/jwt/jwk"

	"github.com/a-novel/authentication/internal/services"
	servicesmocks "github.com/a-novel/authentication/internal/services/mocks"
	"github.com/a-novel/authentication/models"
)

func TestConsumeRefreshToken(t *testing.T) {
	t.Parallel()

	privateRefreshKeys, publicRefreshKeys := generateAuthTokenKeySet(t, 3)

	publicRefreshKeysJSON := lo.Map(publicRefreshKeys, func(item *jwk.Key[ed25519.PublicKey], _ int) *jwa.JWK {
		return item.JWK
	})

	fakePrivateRefreshKey, _, err := jwk.GenerateED25519()
	require.NoError(t, err)

	privateAccessKeys, publicAccessKeys := generateAuthTokenKeySet(t, 3)

	publicAccessKeysJSON := lo.Map(publicAccessKeys, func(item *jwk.Key[ed25519.PublicKey], _ int) *jwa.JWK {
		return item.JWK
	})

	fakePrivateAccessKey, _, err := jwk.GenerateED25519()
	require.NoError(t, err)

	type issueTokenData struct {
		resp string
		err  error
	}

	sampleRefreshToken := mustIssueRefreshToken(t, privateRefreshKeys[0], services.IssueRefreshTokenRequest{
		Claims: &models.AccessTokenClaims{
			UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
			Roles:  []models.Role{"foo", "bar"},
		},
	})

	sampleRefreshTokenParsed, err := jwt.DecodeToken(sampleRefreshToken, new(jwt.SignedTokenDecoder))
	require.NoError(t, err)

	decoded, err := base64.RawURLEncoding.DecodeString(sampleRefreshTokenParsed.Payload)
	require.NoError(t, err)

	var refreshPayload models.RefreshTokenClaims

	require.NoError(t, json.Unmarshal(decoded, &refreshPayload))

	testCases := []struct {
		name string

		request services.ConsumeRefreshTokenRequest

		issueTokenData *issueTokenData

		expect    string
		expectErr error
	}{
		{
			name: "Success",

			request: services.ConsumeRefreshTokenRequest{
				RefreshToken: sampleRefreshToken,
				AccessToken: mustIssueToken(t, privateAccessKeys[0], services.IssueTokenRequest{
					UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:  []models.Role{"foo", "bar"},
				}),
			},

			issueTokenData: &issueTokenData{
				resp: "access_token",
			},

			expect: "access_token",
		},
		{
			name: "IssuedByTheSameRefreshToken",

			request: services.ConsumeRefreshTokenRequest{
				RefreshToken: sampleRefreshToken,
				AccessToken: mustIssueToken(t, privateAccessKeys[0], services.IssueTokenRequest{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:          []models.Role{"foo", "bar"},
					RefreshTokenID: &refreshPayload.Jti,
				}),
			},

			issueTokenData: &issueTokenData{
				resp: "access_token",
			},

			expect: "access_token",
		},
		{
			name: "IssuedByAnotherRefreshToken",

			request: services.ConsumeRefreshTokenRequest{
				RefreshToken: sampleRefreshToken,
				AccessToken: mustIssueToken(t, privateAccessKeys[0], services.IssueTokenRequest{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:          []models.Role{"foo", "bar"},
					RefreshTokenID: lo.ToPtr("00000000-0000-0000-0000-000000000001"),
				}),
			},

			expectErr: services.ErrTokenIssuedWithDifferentRefreshToken,
		},
		{
			name: "MismatchClaims",

			request: services.ConsumeRefreshTokenRequest{
				RefreshToken: sampleRefreshToken,
				AccessToken: mustIssueToken(t, privateAccessKeys[0], services.IssueTokenRequest{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
					Roles:          []models.Role{"foo", "bar"},
					RefreshTokenID: &refreshPayload.Jti,
				}),
			},

			expectErr: services.ErrMismatchRefreshClaims,
		},
		{
			name: "NoAccessToken",

			request: services.ConsumeRefreshTokenRequest{
				RefreshToken: sampleRefreshToken,
			},

			expectErr: models.ErrUnauthorized,
		},
		{
			name: "NoRefreshToken",

			request: services.ConsumeRefreshTokenRequest{
				AccessToken: mustIssueToken(t, privateAccessKeys[0], services.IssueTokenRequest{
					UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:  []models.Role{"foo", "bar"},
				}),
			},

			expectErr: models.ErrUnauthorized,
		},
		{
			name: "InvalidAccessToken",

			request: services.ConsumeRefreshTokenRequest{
				RefreshToken: sampleRefreshToken,
				AccessToken: mustIssueToken(t, fakePrivateAccessKey, services.IssueTokenRequest{
					UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:  []models.Role{"foo", "bar"},
				}),
			},

			expectErr: models.ErrUnauthorized,
		},
		{
			name: "InvalidRefreshToken",

			request: services.ConsumeRefreshTokenRequest{
				RefreshToken: mustIssueRefreshToken(t, fakePrivateRefreshKey, services.IssueRefreshTokenRequest{
					Claims: &models.AccessTokenClaims{
						UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
						Roles:  []models.Role{"foo", "bar"},
					},
				}),
				AccessToken: mustIssueToken(t, privateAccessKeys[0], services.IssueTokenRequest{
					UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:  []models.Role{"foo", "bar"},
				}),
			},

			expectErr: models.ErrUnauthorized,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			source := servicesmocks.NewMockConsumeRefreshTokenSource(t)

			accessTokenKeysSource := jwk.NewED25519PublicSource(jwk.SourceConfig{
				Fetch: func(_ context.Context) ([]*jwa.JWK, error) {
					return publicAccessKeysJSON, nil
				},
			})
			refreshTokenKeysSource := jwk.NewED25519PublicSource(jwk.SourceConfig{
				Fetch: func(_ context.Context) ([]*jwa.JWK, error) { return publicRefreshKeysJSON, nil },
			})

			if testCase.issueTokenData != nil {
				source.
					On("IssueToken", t.Context(), mock.AnythingOfType("IssueTokenRequest")).
					Return(testCase.issueTokenData.resp, testCase.issueTokenData.err)
			}

			service := services.NewConsumeRefreshTokenService(source, accessTokenKeysSource, refreshTokenKeysSource)

			resp, err := service.ConsumeRefreshToken(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			source.AssertExpectations(t)
		})
	}
}
