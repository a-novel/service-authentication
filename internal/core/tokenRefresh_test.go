package core_test

import (
	"encoding/base64"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/jwt/v2"
	"github.com/a-novel-kit/jwt/v2/jwp"
	"github.com/a-novel-kit/jwt/v2/jws"

	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

func TestTokenRefresh(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type daoMock struct {
		resp *dao.Credentials
		err  error
	}

	type revocationExistMock struct {
		jtis []string

		resp bool
		err  error
	}

	type signClaimsMock struct {
		sessionID string

		resp *servicejsonkeys.ClaimsSignResponse
		err  error
	}

	type serviceVerifyClaimsMock struct {
		resp *session.Claims
		err  error
	}

	type serviceVerifyRefreshClaimsMock struct {
		resp *session.RefreshClaims
		err  error
	}

	testCases := []struct {
		name string

		request *core.TokenRefreshRequest

		daoMock                        *daoMock
		revocationExistMock            *revocationExistMock
		signClaimsMock                 *signClaimsMock
		serviceVerifyClaimsMock        *serviceVerifyClaimsMock
		serviceVerifyRefreshClaimsMock *serviceVerifyRefreshClaimsMock

		expect    *core.Token
		expectErr error
		// expectNotErr is a sentinel the returned error must not carry.
		expectNotErr error
	}{
		{
			// A token without a session claim opened its own session, so its JTI stands in for it.
			name: "Success",

			request: &core.TokenRefreshRequest{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			serviceVerifyClaimsMock: &serviceVerifyClaimsMock{
				resp: &session.Claims{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:          []string{"admin"},
					RefreshTokenID: "refresh_token_id",
				},
			},

			serviceVerifyRefreshClaimsMock: &serviceVerifyRefreshClaimsMock{
				resp: &session.RefreshClaims{
					Jti:    "refresh_token_id",
					UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},

			revocationExistMock: &revocationExistMock{jtis: []string{"refresh_token_id", "refresh_token_id"}},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID:   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Role: "admin",
				},
			},

			signClaimsMock: &signClaimsMock{
				sessionID: "refresh_token_id",
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
			// A token of a session another token opened keeps that session.
			name: "Success/Session",

			request: &core.TokenRefreshRequest{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			serviceVerifyClaimsMock: &serviceVerifyClaimsMock{
				resp: &session.Claims{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:          []string{"admin"},
					RefreshTokenID: "refresh_token_id",
					SessionID:      "session_id",
				},
			},

			serviceVerifyRefreshClaimsMock: &serviceVerifyRefreshClaimsMock{
				resp: &session.RefreshClaims{
					Jti:       "refresh_token_id",
					UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					SessionID: "session_id",
				},
			},

			revocationExistMock: &revocationExistMock{jtis: []string{"refresh_token_id", "session_id"}},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID:   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Role: "admin",
				},
			},

			signClaimsMock: &signClaimsMock{
				sessionID: "session_id",
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
			// Signing out from any token of the session stops this one too.
			name: "RevokedSession",

			request: &core.TokenRefreshRequest{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			serviceVerifyClaimsMock: &serviceVerifyClaimsMock{
				resp: &session.Claims{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:          []string{"admin"},
					RefreshTokenID: "refresh_token_id",
					SessionID:      "session_id",
				},
			},

			serviceVerifyRefreshClaimsMock: &serviceVerifyRefreshClaimsMock{
				resp: &session.RefreshClaims{
					Jti:       "refresh_token_id",
					UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					SessionID: "session_id",
				},
			},

			revocationExistMock: &revocationExistMock{jtis: []string{"refresh_token_id", "session_id"}, resp: true},

			expectErr: core.ErrTokenRefreshInvalidRefreshToken,
		},
		{
			name: "SignError",

			request: &core.TokenRefreshRequest{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			serviceVerifyClaimsMock: &serviceVerifyClaimsMock{
				resp: &session.Claims{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:          []string{"admin"},
					RefreshTokenID: "refresh_token_id",
				},
			},

			serviceVerifyRefreshClaimsMock: &serviceVerifyRefreshClaimsMock{
				resp: &session.RefreshClaims{
					Jti:    "refresh_token_id",
					UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},

			revocationExistMock: &revocationExistMock{jtis: []string{"refresh_token_id", "refresh_token_id"}},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID:   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Role: "admin",
				},
			},

			signClaimsMock: &signClaimsMock{
				sessionID: "refresh_token_id",
				err:       errFoo,
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
				resp: &session.Claims{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:          []string{"admin"},
					RefreshTokenID: "refresh_token_id",
				},
			},

			serviceVerifyRefreshClaimsMock: &serviceVerifyRefreshClaimsMock{
				resp: &session.RefreshClaims{
					Jti:    "refresh_token_id",
					UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},

			revocationExistMock: &revocationExistMock{jtis: []string{"refresh_token_id", "refresh_token_id"}},

			daoMock: &daoMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "RevokedRefreshToken",

			request: &core.TokenRefreshRequest{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			serviceVerifyClaimsMock: &serviceVerifyClaimsMock{
				resp: &session.Claims{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:          []string{"admin"},
					RefreshTokenID: "refresh_token_id",
				},
			},

			serviceVerifyRefreshClaimsMock: &serviceVerifyRefreshClaimsMock{
				resp: &session.RefreshClaims{
					Jti:    "refresh_token_id",
					UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},

			revocationExistMock: &revocationExistMock{jtis: []string{"refresh_token_id", "refresh_token_id"}, resp: true},

			expectErr: core.ErrTokenRefreshInvalidRefreshToken,
		},
		{
			name: "RevocationExistError",

			request: &core.TokenRefreshRequest{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			serviceVerifyClaimsMock: &serviceVerifyClaimsMock{
				resp: &session.Claims{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:          []string{"admin"},
					RefreshTokenID: "refresh_token_id",
				},
			},

			serviceVerifyRefreshClaimsMock: &serviceVerifyRefreshClaimsMock{
				resp: &session.RefreshClaims{
					Jti:    "refresh_token_id",
					UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},

			revocationExistMock: &revocationExistMock{jtis: []string{"refresh_token_id", "refresh_token_id"}, err: errFoo},

			expectErr:    errFoo,
			expectNotErr: core.ErrTokenRefreshInvalidRefreshToken,
		},
		{
			name: "VerifyRefreshTokenClaimsError",

			request: &core.TokenRefreshRequest{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			serviceVerifyClaimsMock: &serviceVerifyClaimsMock{
				resp: &session.Claims{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:          []string{"admin"},
					RefreshTokenID: "refresh_token_id",
				},
			},

			serviceVerifyRefreshClaimsMock: &serviceVerifyRefreshClaimsMock{
				err: errFoo,
			},

			expectErr:    errFoo,
			expectNotErr: core.ErrTokenRefreshInvalidRefreshToken,
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

			expectErr:    errFoo,
			expectNotErr: core.ErrTokenRefreshInvalidAccessToken,
		},

		{
			name: "InvalidRefreshToken",

			request: &core.TokenRefreshRequest{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			serviceVerifyClaimsMock: &serviceVerifyClaimsMock{
				resp: &session.Claims{
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
				resp: &session.Claims{
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
			name: "InvalidAccessToken/UnsupportedTokenFormat",

			request: &core.TokenRefreshRequest{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			serviceVerifyClaimsMock: &serviceVerifyClaimsMock{
				err: fmt.Errorf("(HeaderDecoder.Decode) %w: expected at least 2 segments", jwt.ErrUnsupportedTokenFormat),
			},

			expectErr: core.ErrTokenRefreshInvalidAccessToken,
		},
		{
			name: "InvalidAccessToken/TokenTooLarge",

			request: &core.TokenRefreshRequest{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			serviceVerifyClaimsMock: &serviceVerifyClaimsMock{
				err: fmt.Errorf("(Recipient.Consume) %w: 262145 bytes exceeds limit 262144", jwt.ErrTokenTooLarge),
			},

			expectErr: core.ErrTokenRefreshInvalidAccessToken,
		},
		{
			name: "InvalidAccessToken/MismatchRecipientPlugin",

			request: &core.TokenRefreshRequest{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			serviceVerifyClaimsMock: &serviceVerifyClaimsMock{
				err: fmt.Errorf(
					"(Recipient.Consume) %w: %w: no compatible plugin found",
					jwt.ErrTokenRejected, jwt.ErrMismatchRecipientPlugin,
				),
			},

			expectErr: core.ErrTokenRefreshInvalidAccessToken,
		},
		{
			name: "InvalidAccessToken/MissingCritHeader",

			request: &core.TokenRefreshRequest{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			serviceVerifyClaimsMock: &serviceVerifyClaimsMock{
				err: fmt.Errorf("(Recipient.Consume) check crit: %w: b64", jwt.ErrMissingCritHeader),
			},

			expectErr: core.ErrTokenRefreshInvalidAccessToken,
		},
		{
			name: "InvalidAccessToken/UnsupportedCritHeader",

			request: &core.TokenRefreshRequest{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			serviceVerifyClaimsMock: &serviceVerifyClaimsMock{
				err: fmt.Errorf(
					"(Recipient.Consume) check crit: %w: \"b64\" is not understood by this recipient",
					jwt.ErrUnsupportedCritHeader,
				),
			},

			expectErr: core.ErrTokenRefreshInvalidAccessToken,
		},
		{
			name: "InvalidAccessToken/ReplicatedClaimMismatch",

			request: &core.TokenRefreshRequest{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			serviceVerifyClaimsMock: &serviceVerifyClaimsMock{
				err: fmt.Errorf("(Recipient.Consume) %w: %w: iss", jwt.ErrTokenRejected, jwt.ErrReplicatedClaimMismatch),
			},

			expectErr: core.ErrTokenRefreshInvalidAccessToken,
		},
		{
			name: "InvalidRefreshToken/ReplicatedClaimMismatch",

			request: &core.TokenRefreshRequest{
				AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
				RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
			},

			serviceVerifyClaimsMock: &serviceVerifyClaimsMock{
				resp: &session.Claims{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:          []string{"admin"},
					RefreshTokenID: "refresh_token_id",
				},
			},

			serviceVerifyRefreshClaimsMock: &serviceVerifyRefreshClaimsMock{
				err: fmt.Errorf("(Recipient.Consume) %w: %w: sub", jwt.ErrTokenRejected, jwt.ErrReplicatedClaimMismatch),
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
				resp: &session.Claims{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:          []string{"admin"},
					RefreshTokenID: "refresh_token_id",
				},
			},

			serviceVerifyRefreshClaimsMock: &serviceVerifyRefreshClaimsMock{
				resp: &session.RefreshClaims{
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
				resp: &session.Claims{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:          []string{"admin"},
					RefreshTokenID: "refresh_token_id",
				},
			},

			serviceVerifyRefreshClaimsMock: &serviceVerifyRefreshClaimsMock{
				resp: &session.RefreshClaims{
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
			mockDaoRevocationExist := coremocks.NewMockTokenRefreshDaoRefreshTokenRevocationExist(t)
			serviceSignClaims := coremocks.NewMockTokenRefreshServiceSignClaims(t)
			serviceVerifyClaims := coremocks.NewMockTokenRefreshServiceVerifyClaims(t)
			serviceVerifyRefreshClaims := coremocks.NewMockTokenRefreshServiceVerifyRefreshClaims(t)

			if testCase.serviceVerifyClaimsMock != nil {
				verifyClaimsRequest := &servicejsonkeys.VerifyClaimsRequest{
					Usage:       servicejsonkeys.KeyUsageAuth,
					AccessToken: testCase.request.AccessToken,
					Options:     &servicejsonkeys.VerifyClaimsOptions{IgnoreExpired: true},
				}

				serviceVerifyClaims.EXPECT().
					VerifyClaims(mock.Anything, verifyClaimsRequest).
					Return(testCase.serviceVerifyClaimsMock.resp, testCase.serviceVerifyClaimsMock.err)
			}

			if testCase.serviceVerifyRefreshClaimsMock != nil {
				verifyClaimsRequest := &servicejsonkeys.VerifyClaimsRequest{
					Usage:       servicejsonkeys.KeyUsageAuthRefresh,
					AccessToken: testCase.request.RefreshToken,
				}

				serviceVerifyRefreshClaims.EXPECT().
					VerifyClaims(mock.Anything, verifyClaimsRequest).
					Return(testCase.serviceVerifyRefreshClaimsMock.resp, testCase.serviceVerifyRefreshClaimsMock.err)
			}

			if testCase.revocationExistMock != nil {
				mockDaoRevocationExist.EXPECT().
					Exec(mock.Anything, &dao.RefreshTokenRevocationExistRequest{JTIs: testCase.revocationExistMock.jtis}).
					Return(testCase.revocationExistMock.resp, testCase.revocationExistMock.err)
			}

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, &dao.CredentialsSelectRequest{
						ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					}).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			if testCase.signClaimsMock != nil {
				accessTokenClaims := &session.Claims{
					UserID:         testCase.serviceVerifyClaimsMock.resp.UserID,
					Roles:          []string{testCase.daoMock.resp.Role},
					RefreshTokenID: testCase.serviceVerifyRefreshClaimsMock.resp.Jti,
					SessionID:      testCase.signClaimsMock.sessionID,
				}
				claimsSignRequest := &servicejsonkeys.ClaimsSignRequest{
					Usage:   servicejsonkeys.KeyUsageAuth,
					Payload: lo.Must(grpcf.MarshalJSONAsAny(accessTokenClaims)),
				}

				serviceSignClaims.EXPECT().
					ClaimsSign(mock.Anything, claimsSignRequest).
					Return(testCase.signClaimsMock.resp, testCase.signClaimsMock.err)
			}

			service := core.NewTokenRefresh(
				mockDao,
				mockDaoRevocationExist,
				serviceSignClaims,
				serviceVerifyClaims,
				serviceVerifyRefreshClaims,
			)

			resp, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			if testCase.expectNotErr != nil {
				require.NotErrorIs(t, err, testCase.expectNotErr)
			}

			mockDao.AssertExpectations(t)
			mockDaoRevocationExist.AssertExpectations(t)
			serviceSignClaims.AssertExpectations(t)
			serviceVerifyClaims.AssertExpectations(t)
			serviceVerifyRefreshClaims.AssertExpectations(t)
		})
	}
}
