package services_test

import (
	"encoding/base64"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/golib/grpcf"
	jkpkg "github.com/a-novel/service-json-keys/v2/pkg"

	"github.com/a-novel-kit/jwt/jws"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/services"
	servicesmocks "github.com/a-novel/service-authentication/internal/services/mocks"
)

func TestTokenRefresh(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type repositoryMock struct {
		resp *dao.Credentials
		err  error
	}

	type signClaimsMock struct {
		resp *jkpkg.ClaimsSignResponse
		err  error
	}

	type serviceVerifyClaimsMock struct {
		resp *services.AccessTokenClaims
		err  error
	}

	type serviceVerifyRefreshClaimsMock struct {
		resp *services.RefreshTokenClaims
		err  error
	}

	testCases := []struct {
		name string

		request *services.TokenRefreshRequest

		repositoryMock                 *repositoryMock
		signClaimsMock                 *signClaimsMock
		serviceVerifyClaimsMock        *serviceVerifyClaimsMock
		serviceVerifyRefreshClaimsMock *serviceVerifyRefreshClaimsMock

		expect    *services.Token
		expectErr error
	}{
		{
			name: "Success",

			request: &services.TokenRefreshRequest{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			serviceVerifyClaimsMock: &serviceVerifyClaimsMock{
				resp: &services.AccessTokenClaims{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:          []string{"admin"},
					RefreshTokenID: "refresh_token_id",
				},
			},

			serviceVerifyRefreshClaimsMock: &serviceVerifyRefreshClaimsMock{
				resp: &services.RefreshTokenClaims{
					Jti:    "refresh_token_id",
					UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},

			repositoryMock: &repositoryMock{
				resp: &dao.Credentials{
					ID:   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Role: "admin",
				},
			},

			signClaimsMock: &signClaimsMock{
				resp: &jkpkg.ClaimsSignResponse{
					Token: base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				},
			},

			expect: &services.Token{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},
		},

		{
			name: "SignError",

			request: &services.TokenRefreshRequest{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			serviceVerifyClaimsMock: &serviceVerifyClaimsMock{
				resp: &services.AccessTokenClaims{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:          []string{"admin"},
					RefreshTokenID: "refresh_token_id",
				},
			},

			serviceVerifyRefreshClaimsMock: &serviceVerifyRefreshClaimsMock{
				resp: &services.RefreshTokenClaims{
					Jti:    "refresh_token_id",
					UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},

			repositoryMock: &repositoryMock{
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

			request: &services.TokenRefreshRequest{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			serviceVerifyClaimsMock: &serviceVerifyClaimsMock{
				resp: &services.AccessTokenClaims{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:          []string{"admin"},
					RefreshTokenID: "refresh_token_id",
				},
			},

			serviceVerifyRefreshClaimsMock: &serviceVerifyRefreshClaimsMock{
				resp: &services.RefreshTokenClaims{
					Jti:    "refresh_token_id",
					UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},

			repositoryMock: &repositoryMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "VerifyRefreshTokenClaimsError",

			request: &services.TokenRefreshRequest{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			serviceVerifyClaimsMock: &serviceVerifyClaimsMock{
				resp: &services.AccessTokenClaims{
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

			request: &services.TokenRefreshRequest{
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

			request: &services.TokenRefreshRequest{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			serviceVerifyClaimsMock: &serviceVerifyClaimsMock{
				resp: &services.AccessTokenClaims{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:          []string{"admin"},
					RefreshTokenID: "refresh_token_id",
				},
			},

			serviceVerifyRefreshClaimsMock: &serviceVerifyRefreshClaimsMock{
				err: jws.ErrInvalidSignature,
			},

			expectErr: services.ErrTokenRefreshInvalidRefreshToken,
		},
		{
			name: "InvalidAccessToken",

			request: &services.TokenRefreshRequest{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			serviceVerifyClaimsMock: &serviceVerifyClaimsMock{
				err: jws.ErrInvalidSignature,
			},

			expectErr: services.ErrTokenRefreshInvalidAccessToken,
		},

		{
			name: "NoRefreshToken",

			request: &services.TokenRefreshRequest{
				AccessToken: base64.RawURLEncoding.EncodeToString([]byte("access-token")),
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "NoAccessToken",

			request: &services.TokenRefreshRequest{
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "NotSameUser",

			request: &services.TokenRefreshRequest{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			serviceVerifyClaimsMock: &serviceVerifyClaimsMock{
				resp: &services.AccessTokenClaims{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:          []string{"admin"},
					RefreshTokenID: "refresh_token_id",
				},
			},

			serviceVerifyRefreshClaimsMock: &serviceVerifyRefreshClaimsMock{
				resp: &services.RefreshTokenClaims{
					Jti:    "refresh_token_id",
					UserID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				},
			},

			expectErr: services.ErrTokenRefreshMismatchClaims,
		},
		{
			name: "NotSameRefreshToken",

			request: &services.TokenRefreshRequest{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			serviceVerifyClaimsMock: &serviceVerifyClaimsMock{
				resp: &services.AccessTokenClaims{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:          []string{"admin"},
					RefreshTokenID: "refresh_token_id",
				},
			},

			serviceVerifyRefreshClaimsMock: &serviceVerifyRefreshClaimsMock{
				resp: &services.RefreshTokenClaims{
					Jti:    "other_refresh_token_id",
					UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},

			expectErr: services.ErrTokenRefreshMismatchSource,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			repository := servicesmocks.NewMockTokenRefreshRepository(t)
			serviceSignClaims := servicesmocks.NewMockTokenRefreshServiceSignClaims(t)
			serviceVerifyClaims := servicesmocks.NewMockTokenRefreshServiceVerifyClaims(t)
			serviceVerifyRefreshClaims := servicesmocks.NewMockTokenRefreshServiceVerifyRefreshClaims(t)

			if testCase.serviceVerifyClaimsMock != nil {
				serviceVerifyClaims.EXPECT().
					VerifyClaims(
						mock.Anything,
						&jkpkg.VerifyClaimsRequest{
							Usage:       jkpkg.KeyUsageAuth,
							AccessToken: testCase.request.AccessToken,
							Options:     &jkpkg.VerifyClaimsOptions{IgnoreExpired: true},
						},
					).
					Return(testCase.serviceVerifyClaimsMock.resp, testCase.serviceVerifyClaimsMock.err)
			}

			if testCase.serviceVerifyRefreshClaimsMock != nil {
				serviceVerifyRefreshClaims.EXPECT().
					VerifyClaims(
						mock.Anything,
						&jkpkg.VerifyClaimsRequest{
							Usage:       jkpkg.KeyUsageAuthRefresh,
							AccessToken: testCase.request.RefreshToken,
						},
					).
					Return(testCase.serviceVerifyRefreshClaimsMock.resp, testCase.serviceVerifyRefreshClaimsMock.err)
			}

			if testCase.repositoryMock != nil {
				repository.EXPECT().
					Exec(mock.Anything, &dao.CredentialsSelectRequest{
						ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					}).
					Return(testCase.repositoryMock.resp, testCase.repositoryMock.err)
			}

			if testCase.signClaimsMock != nil {
				serviceSignClaims.EXPECT().
					ClaimsSign(
						mock.Anything,
						&jkpkg.ClaimsSignRequest{
							Usage: jkpkg.KeyUsageAuth,
							Payload: lo.Must(grpcf.InterfaceToProtoAny(&services.AccessTokenClaims{
								UserID:         testCase.serviceVerifyClaimsMock.resp.UserID,
								Roles:          []string{testCase.repositoryMock.resp.Role},
								RefreshTokenID: testCase.serviceVerifyRefreshClaimsMock.resp.Jti,
							})),
						},
					).
					Return(testCase.signClaimsMock.resp, testCase.signClaimsMock.err)
			}

			service := services.NewTokenRefresh(
				repository,
				serviceSignClaims,
				serviceVerifyClaims,
				serviceVerifyRefreshClaims,
			)

			resp, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			repository.AssertExpectations(t)
			serviceSignClaims.AssertExpectations(t)
			serviceVerifyClaims.AssertExpectations(t)
			serviceVerifyRefreshClaims.AssertExpectations(t)
		})
	}
}
