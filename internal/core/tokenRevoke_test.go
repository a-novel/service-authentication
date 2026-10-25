package core_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/jwt/v2/jws"

	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

func TestTokenRevoke(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	otherUserID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	leeway := 5 * time.Minute
	dayLater := time.Now().Add(24 * time.Hour).Unix()
	dayAgo := time.Now().Add(-24 * time.Hour).Unix()

	type serviceVerifyRefreshClaimsMock struct {
		resp *session.RefreshClaims
		err  error
	}

	type daoMock struct {
		expiresAt time.Time
		err       error
	}

	testCases := []struct {
		name string

		request *core.TokenRevokeRequest

		serviceVerifyRefreshClaimsMock *serviceVerifyRefreshClaimsMock
		daoMock                        *daoMock

		expectErr error
		// expectNotErr is a sentinel the returned error must not carry.
		expectNotErr error
	}{
		{
			name: "Success",

			request: &core.TokenRevokeRequest{UserID: userID, RefreshToken: "refresh-token"},

			serviceVerifyRefreshClaimsMock: &serviceVerifyRefreshClaimsMock{
				resp: &session.RefreshClaims{
					Jti:    "refresh-token-id",
					Exp:    dayLater,
					UserID: userID,
				},
			},

			daoMock: &daoMock{expiresAt: time.Unix(dayLater, 0).Add(leeway)},
		},
		{
			// An expired refresh token no longer renews the session, and still revokes so a retry
			// after expiry succeeds.
			name: "Success/ExpiredToken",

			request: &core.TokenRevokeRequest{UserID: userID, RefreshToken: "refresh-token"},

			serviceVerifyRefreshClaimsMock: &serviceVerifyRefreshClaimsMock{
				resp: &session.RefreshClaims{
					Jti:    "refresh-token-id",
					Exp:    dayAgo,
					UserID: userID,
				},
			},

			daoMock: &daoMock{expiresAt: time.Unix(dayAgo, 0).Add(leeway)},
		},
		{
			// A refresh token another user holds stays alive, whoever sends it.
			name: "Error/MismatchClaims",

			request: &core.TokenRevokeRequest{UserID: userID, RefreshToken: "refresh-token"},

			serviceVerifyRefreshClaimsMock: &serviceVerifyRefreshClaimsMock{
				resp: &session.RefreshClaims{Jti: "refresh-token-id", Exp: dayLater, UserID: otherUserID},
			},

			expectErr: core.ErrTokenRevokeMismatchClaims,
		},
		{
			name: "Error/InvalidRefreshToken",

			request: &core.TokenRevokeRequest{UserID: userID, RefreshToken: "refresh-token"},

			serviceVerifyRefreshClaimsMock: &serviceVerifyRefreshClaimsMock{
				err: jws.ErrInvalidSignature,
			},

			expectErr: core.ErrTokenRevokeInvalidRefreshToken,
		},
		{
			// A verifier failure that is not a rejection, such as an unreachable key source, is a
			// server fault.
			name: "Error/VerifyRefreshClaims",

			request: &core.TokenRevokeRequest{UserID: userID, RefreshToken: "refresh-token"},

			serviceVerifyRefreshClaimsMock: &serviceVerifyRefreshClaimsMock{
				err: errFoo,
			},

			expectErr:    errFoo,
			expectNotErr: core.ErrTokenRevokeInvalidRefreshToken,
		},
		{
			name: "Error/Dao",

			request: &core.TokenRevokeRequest{UserID: userID, RefreshToken: "refresh-token"},

			serviceVerifyRefreshClaimsMock: &serviceVerifyRefreshClaimsMock{
				resp: &session.RefreshClaims{
					Jti:    "refresh-token-id",
					Exp:    dayLater,
					UserID: userID,
				},
			},

			daoMock: &daoMock{expiresAt: time.Unix(dayLater, 0).Add(leeway), err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/NoRefreshToken",

			request: &core.TokenRevokeRequest{},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/RefreshTokenTooLong",

			request: &core.TokenRevokeRequest{RefreshToken: strings.Repeat("a", 1025)},

			expectErr: core.ErrInvalidRequest,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockDao := coremocks.NewMockTokenRevokeDao(t)
			serviceVerifyRefreshClaims := coremocks.NewMockTokenRevokeServiceVerifyRefreshClaims(t)

			if testCase.serviceVerifyRefreshClaimsMock != nil {
				verifyClaimsRequest := &servicejsonkeys.VerifyClaimsRequest{
					Usage:       servicejsonkeys.KeyUsageAuthRefresh,
					AccessToken: testCase.request.RefreshToken,
					Options:     &servicejsonkeys.VerifyClaimsOptions{IgnoreExpired: true},
				}

				serviceVerifyRefreshClaims.EXPECT().
					VerifyClaims(mock.Anything, verifyClaimsRequest).
					Return(testCase.serviceVerifyRefreshClaimsMock.resp, testCase.serviceVerifyRefreshClaimsMock.err)
			}

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(request *dao.RefreshTokenRevocationInsertRequest) bool {
						return assert.Equal(t, "refresh-token-id", request.JTI) &&
							assert.True(t, testCase.daoMock.expiresAt.Equal(request.ExpiresAt)) &&
							assert.WithinDuration(t, time.Now(), request.Now, time.Minute)
					})).
					Return(testCase.daoMock.err)
			}

			service := core.NewTokenRevoke(mockDao, serviceVerifyRefreshClaims, leeway)

			err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)

			if testCase.expectNotErr != nil {
				require.NotErrorIs(t, err, testCase.expectNotErr)
			}

			mockDao.AssertExpectations(t)
			serviceVerifyRefreshClaims.AssertExpectations(t)
		})
	}
}
