package core_test

import (
	"encoding/base64"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/jwt/v2/jwp"
	"github.com/a-novel-kit/jwt/v2/jws"

	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
)

func TestTokenRefresh(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type daoMock struct {
		resp *dao.Credentials
		err  error
	}

	type signClaimsMock struct {
		resp *servicejsonkeys.ClaimsSignResponse
		err  error
	}

	type serviceVerifyClaimsMock struct {
		resp *core.AccessTokenClaims
		err  error
	}

	type serviceVerifyRefreshClaimsMock struct {
		resp *core.RefreshTokenClaims
		err  error
	}

	testCases := []struct {
		name string

		request *core.TokenRefreshRequest

		daoMock                        *daoMock
		signClaimsMock                 *signClaimsMock
		serviceVerifyClaimsMock        *serviceVerifyClaimsMock
		serviceVerifyRefreshClaimsMock *serviceVerifyRefreshClaimsMock

		expect    *core.Token
		expectErr error
	}{
		{
			name: "Success",

			request: &core.TokenRefreshRequest{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			serviceVerifyClaimsMock: &serviceVerifyClaimsMock{
				resp: &core.AccessTokenClaims{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:          []string{"admin"},
					RefreshTokenID: "refresh_token_id",
				},
			},

			serviceVerifyRefreshClaimsMock: &serviceVerifyRefreshClaimsMock{
				resp: &core.RefreshTokenClaims{
					Jti:    "refresh_token_id",
					UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID:   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Role: "admin",
				},
			},

			signClaimsMock: &signClaimsMock{
				resp: &servicejsonkeys.ClaimsSignResponse{
					Token: base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				},
			},

			expect: &core.Token{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},
		},

		{
			name: "SignError",

			request: &core.TokenRefreshRequest{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			serviceVerifyClaimsMock: &serviceVerifyClaimsMock{
				resp: &core.AccessTokenClaims{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:          []string{"admin"},
					RefreshTokenID: "refresh_token_id",
				},
			},

			serviceVerifyRefreshClaimsMock: &serviceVerifyRefreshClaimsMock{
				resp: &core.RefreshTokenClaims{
					Jti:    "refresh_token_id",
					UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID:   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Role: "admin",
				},
			},

			signClaimsMock: &signClaimsMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "SelectCredentialsError",

			request: &core.TokenRefreshRequest{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			serviceVerifyClaimsMock: &serviceVerifyClaimsMock{
				resp: &core.AccessTokenClaims{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:          []string{"admin"},
					RefreshTokenID: "refresh_token_id",
				},
			},

			serviceVerifyRefreshClaimsMock: &serviceVerifyRefreshClaimsMock{
				resp: &core.RefreshTokenClaims{
					Jti:    "refresh_token_id",
					UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},

			daoMock: &daoMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "VerifyRefreshTokenClaimsError",

			request: &core.TokenRefreshRequest{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			serviceVerifyClaimsMock: &serviceVerifyClaimsMock{
				resp: &core.AccessTokenClaims{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:          []string{"admin"},
					RefreshTokenID: "refresh_token_id",
				},
			},

			serviceVerifyRefreshClaimsMock: &serviceVerifyRefreshClaimsMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "VerifyClaimsError",

			request: &core.TokenRefreshRequest{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			serviceVerifyClaimsMock: &serviceVerifyClaimsMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},

		{
			name: "InvalidRefreshToken",

			request: &core.TokenRefreshRequest{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			serviceVerifyClaimsMock: &serviceVerifyClaimsMock{
				resp: &core.AccessTokenClaims{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:          []string{"admin"},
					RefreshTokenID: "refresh_token_id",
				},
			},

			serviceVerifyRefreshClaimsMock: &serviceVerifyRefreshClaimsMock{
				err: jws.ErrInvalidSignature,
			},

			expectErr: core.ErrTokenRefreshInvalidRefreshToken,
		},
		{
			name: "InvalidAccessToken",

			request: &core.TokenRefreshRequest{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			serviceVerifyClaimsMock: &serviceVerifyClaimsMock{
				err: jws.ErrInvalidSignature,
			},

			expectErr: core.ErrTokenRefreshInvalidAccessToken,
		},
		{
			// Audience/issuer/subject mismatch on the access token surfaces as
			// jwp.ErrInvalidClaims. (Expiry is ignored on the access token.)
			name: "InvalidAccessTokenClaims",

			request: &core.TokenRefreshRequest{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			serviceVerifyClaimsMock: &serviceVerifyClaimsMock{
				err: jwp.ErrInvalidClaims,
			},

			expectErr: core.ErrTokenRefreshInvalidAccessToken,
		},
		{
			// Refresh-token expiry, audience, issuer or subject mismatch all
			// surface through the verifier as jwp.ErrInvalidClaims.
			name: "InvalidRefreshTokenClaims",

			request: &core.TokenRefreshRequest{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			serviceVerifyClaimsMock: &serviceVerifyClaimsMock{
				resp: &core.AccessTokenClaims{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:          []string{"admin"},
					RefreshTokenID: "refresh_token_id",
				},
			},

			serviceVerifyRefreshClaimsMock: &serviceVerifyRefreshClaimsMock{
				err: jwp.ErrInvalidClaims,
			},

			expectErr: core.ErrTokenRefreshInvalidRefreshToken,
		},

		{
			name: "NoRefreshToken",

			request: &core.TokenRefreshRequest{
				AccessToken: base64.RawURLEncoding.EncodeToString([]byte("access-token")),
			},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "NoAccessToken",

			request: &core.TokenRefreshRequest{
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "NotSameUser",

			request: &core.TokenRefreshRequest{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			serviceVerifyClaimsMock: &serviceVerifyClaimsMock{
				resp: &core.AccessTokenClaims{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:          []string{"admin"},
					RefreshTokenID: "refresh_token_id",
				},
			},

			serviceVerifyRefreshClaimsMock: &serviceVerifyRefreshClaimsMock{
				resp: &core.RefreshTokenClaims{
					Jti:    "refresh_token_id",
					UserID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				},
			},

			expectErr: core.ErrTokenRefreshMismatchClaims,
		},
		{
			name: "NotSameRefreshToken",

			request: &core.TokenRefreshRequest{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			serviceVerifyClaimsMock: &serviceVerifyClaimsMock{
				resp: &core.AccessTokenClaims{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:          []string{"admin"},
					RefreshTokenID: "refresh_token_id",
				},
			},

			serviceVerifyRefreshClaimsMock: &serviceVerifyRefreshClaimsMock{
				resp: &core.RefreshTokenClaims{
					Jti:    "other_refresh_token_id",
					UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},

			expectErr: core.ErrTokenRefreshMismatchSource,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockDao := coremocks.NewMockTokenRefreshDao(t)
			serviceSignClaims := coremocks.NewMockTokenRefreshServiceSignClaims(t)
			serviceVerifyClaims := coremocks.NewMockTokenRefreshServiceVerifyClaims(t)
			serviceVerifyRefreshClaims := coremocks.NewMockTokenRefreshServiceVerifyRefreshClaims(t)

			if testCase.serviceVerifyClaimsMock != nil {
				serviceVerifyClaims.EXPECT().
					VerifyClaims(
						mock.Anything,
						&servicejsonkeys.VerifyClaimsRequest{
							Usage:       servicejsonkeys.KeyUsageAuth,
							AccessToken: testCase.request.AccessToken,
							Options:     &servicejsonkeys.VerifyClaimsOptions{IgnoreExpired: true},
						},
					).
					Return(testCase.serviceVerifyClaimsMock.resp, testCase.serviceVerifyClaimsMock.err)
			}

			if testCase.serviceVerifyRefreshClaimsMock != nil {
				serviceVerifyRefreshClaims.EXPECT().
					VerifyClaims(
						mock.Anything,
						&servicejsonkeys.VerifyClaimsRequest{
							Usage:       servicejsonkeys.KeyUsageAuthRefresh,
							AccessToken: testCase.request.RefreshToken,
						},
					).
					Return(testCase.serviceVerifyRefreshClaimsMock.resp, testCase.serviceVerifyRefreshClaimsMock.err)
			}

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, &dao.CredentialsSelectRequest{
						ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					}).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			if testCase.signClaimsMock != nil {
				serviceSignClaims.EXPECT().
					ClaimsSign(
						mock.Anything,
						&servicejsonkeys.ClaimsSignRequest{
							Usage: servicejsonkeys.KeyUsageAuth,
							Payload: lo.Must(grpcf.MarshalJSONAsAny(&core.AccessTokenClaims{
								UserID:         testCase.serviceVerifyClaimsMock.resp.UserID,
								Roles:          []string{testCase.daoMock.resp.Role},
								RefreshTokenID: testCase.serviceVerifyRefreshClaimsMock.resp.Jti,
							})),
						},
					).
					Return(testCase.signClaimsMock.resp, testCase.signClaimsMock.err)
			}

			service := core.NewTokenRefresh(
				mockDao,
				serviceSignClaims,
				serviceVerifyClaims,
				serviceVerifyRefreshClaims,
			)

			resp, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			mockDao.AssertExpectations(t)
			serviceSignClaims.AssertExpectations(t)
			serviceVerifyClaims.AssertExpectations(t)
			serviceVerifyRefreshClaims.AssertExpectations(t)
		})
	}
}
