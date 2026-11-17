package core_test

import (
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/transaction/transactiontest"
	"github.com/a-novel-kit/jwt/v2/jws"

	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

func TestTokenRefresh(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	hourLater := time.Now().Add(time.Hour).Unix()

	// Session IDs are the JTIs of the refresh tokens that open them, UUIDs.
	refreshTokenID := "00000000-0000-0000-0000-0000000000a1"
	sessionID := "00000000-0000-0000-0000-0000000000a2"

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	visitor := core.Visitor{UserAgent: "Firefox", IPAddress: "192.0.2.1", Region: "FR", City: "Paris"}
	request := &core.TokenRefreshRequest{
		AccessToken:  base64.RawURLEncoding.EncodeToString([]byte("access-token")),
		RefreshToken: base64.RawURLEncoding.EncodeToString([]byte("refresh_token")),
		Visitor:      visitor,
	}

	// A token without a session claim opened its own session, so its JTI stands in for it.
	accessClaims := &session.Claims{UserID: &userID, Roles: []string{"auth:user"}, RefreshTokenID: refreshTokenID}
	refreshClaims := &session.RefreshClaims{Jti: refreshTokenID, Exp: hourLater, UserID: userID}
	// The renewal reloads the account, so the new token carries the role it holds now.
	credentials := &dao.Credentials{ID: userID, Role: "auth:admin"}

	type respMock[T any] struct {
		resp T
		err  error
	}

	type revocationMock struct {
		jtis []string
	}

	type signClaimsMock struct {
		sessionID string
		err       error
	}

	type testCase struct {
		name string

		request *core.TokenRefreshRequest

		verifyClaimsMock        *respMock[*session.Claims]
		verifyRefreshClaimsMock *respMock[*session.RefreshClaims]
		revocationMock          *revocationMock
		daoMock                 *respMock[*dao.Credentials]
		signClaimsMock          *signClaimsMock
		// consentMock answers the locked consent read of a consenting account.
		consentMock   *respMock[bool]
		sessionUpsert *dao.SessionUpsertRequest

		expect    *core.Token
		expectErr error
		// expectNotErr is a sentinel the returned error must not carry.
		expectNotErr error
	}

	success := &testCase{
		name: "Success",

		request: request,

		verifyClaimsMock:        &respMock[*session.Claims]{resp: accessClaims},
		verifyRefreshClaimsMock: &respMock[*session.RefreshClaims]{resp: refreshClaims},
		revocationMock:          &revocationMock{jtis: []string{refreshTokenID, refreshTokenID}},
		daoMock:                 &respMock[*dao.Credentials]{resp: credentials},
		signClaimsMock:          &signClaimsMock{sessionID: refreshTokenID},
		// Without consent, the renewal writes no location, and erases any the row holds.
		sessionUpsert: &dao.SessionUpsertRequest{
			ID:        uuid.MustParse(refreshTokenID),
			UserID:    userID,
			UserAgent: "Firefox",
			ExpiresAt: time.Unix(hourLater, 0),
		},

		// The refresh token does not change.
		expect: &core.Token{AccessToken: "new-access-token", RefreshToken: request.RefreshToken},
	}

	testCases := []*testCase{
		success,
		changed(*success, func(variant *testCase) {
			// A token of a session another token opened keeps that session, and records the location
			// under the account's consent.
			variant.name = "Success/Session"
			variant.verifyClaimsMock = &respMock[*session.Claims]{
				resp: changed(*accessClaims, func(claims *session.Claims) { claims.SessionID = sessionID }),
			}
			variant.verifyRefreshClaimsMock = &respMock[*session.RefreshClaims]{
				resp: changed(*refreshClaims, func(claims *session.RefreshClaims) { claims.SessionID = sessionID }),
			}
			variant.revocationMock = &revocationMock{jtis: []string{refreshTokenID, sessionID}}
			variant.daoMock = &respMock[*dao.Credentials]{
				resp: changed(*credentials, func(credentials *dao.Credentials) { credentials.LocationConsent = true }),
			}
			variant.signClaimsMock = &signClaimsMock{sessionID: sessionID}
			variant.consentMock = &respMock[bool]{resp: true}
			variant.sessionUpsert = changed(*success.sessionUpsert, func(request *dao.SessionUpsertRequest) {
				request.ID = uuid.MustParse(sessionID)
				request.IPAddress, request.Region, request.City = "192.0.2.1", "FR", "Paris"
			})
		}),
		{
			name: "Error/NoAccessToken",

			request: changed(*request, func(request *core.TokenRefreshRequest) { request.AccessToken = "" }),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NoRefreshToken",

			request: changed(*request, func(request *core.TokenRefreshRequest) { request.RefreshToken = "" }),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidAccessToken",

			request: request,

			verifyClaimsMock: &respMock[*session.Claims]{err: jws.ErrInvalidSignature},

			expectErr: core.ErrTokenRefreshInvalidAccessToken,
		},
		{
			// A verifier failure that is not a rejection, such as an unreachable key source, is a fault.
			name: "Error/VerifyAccessToken",

			request: request,

			verifyClaimsMock: &respMock[*session.Claims]{err: errFoo},

			expectErr:    errFoo,
			expectNotErr: core.ErrTokenRefreshInvalidAccessToken,
		},
		{
			name: "Error/InvalidRefreshToken",

			request: request,

			verifyClaimsMock:        &respMock[*session.Claims]{resp: accessClaims},
			verifyRefreshClaimsMock: &respMock[*session.RefreshClaims]{err: jws.ErrInvalidSignature},

			expectErr: core.ErrTokenRefreshInvalidRefreshToken,
		},
		{
			name: "Error/VerifyRefreshToken",

			request: request,

			verifyClaimsMock:        &respMock[*session.Claims]{resp: accessClaims},
			verifyRefreshClaimsMock: &respMock[*session.RefreshClaims]{err: errFoo},

			expectErr:    errFoo,
			expectNotErr: core.ErrTokenRefreshInvalidRefreshToken,
		},
		{
			name: "Error/MismatchClaims",

			request: request,

			verifyClaimsMock: &respMock[*session.Claims]{resp: accessClaims},
			verifyRefreshClaimsMock: &respMock[*session.RefreshClaims]{
				resp: changed(*refreshClaims, func(claims *session.RefreshClaims) {
					claims.UserID = uuid.MustParse("00000000-0000-0000-0000-000000000002")
				}),
			},

			expectErr: core.ErrTokenRefreshMismatchClaims,
		},
		{
			name: "Error/MismatchSource",

			request: request,

			verifyClaimsMock: &respMock[*session.Claims]{resp: accessClaims},
			verifyRefreshClaimsMock: &respMock[*session.RefreshClaims]{
				resp: changed(*refreshClaims, func(claims *session.RefreshClaims) {
					claims.Jti = "00000000-0000-0000-0000-0000000000a3"
				}),
			},

			expectErr: core.ErrTokenRefreshMismatchSource,
		},
		{
			// A client's refresh token renews at the token endpoint alone, where the client
			// authenticates (RFC 6749 §10.4).
			name: "Error/ClientToken",

			request: request,

			verifyClaimsMock: &respMock[*session.Claims]{resp: accessClaims},
			verifyRefreshClaimsMock: &respMock[*session.RefreshClaims]{
				resp: changed(*refreshClaims, func(claims *session.RefreshClaims) { claims.ClientID = "studio" }),
			},

			expectErr: core.ErrTokenRefreshInvalidRefreshToken,
		},
		{
			// A session that ended renews no more, though the verifier grants its token a leeway.
			name: "Error/SessionEnded",

			request: request,

			verifyClaimsMock: &respMock[*session.Claims]{resp: accessClaims},
			verifyRefreshClaimsMock: &respMock[*session.RefreshClaims]{
				resp: changed(*refreshClaims, func(claims *session.RefreshClaims) { claims.Exp = time.Now().Unix() }),
			},

			expectErr: core.ErrTokenRefreshInvalidRefreshToken,
		},
		changed(*success, func(variant *testCase) {
			variant.name = "Error/SelectCredentials"
			variant.daoMock = &respMock[*dao.Credentials]{err: errFoo}
			variant.signClaimsMock, variant.sessionUpsert = nil, nil
			variant.expect, variant.expectErr = nil, errFoo
		}),
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockDao := coremocks.NewMockTokenRefreshDao(t)
			mockDaoRevocationExist := coremocks.NewMockTokenRefreshDaoRefreshTokenRevocationExist(t)
			serviceSignClaims := coremocks.NewMockTokenRefreshServiceSignClaims(t)
			serviceVerifyClaims := coremocks.NewMockTokenRefreshServiceVerifyClaims(t)
			serviceVerifyRefreshClaims := coremocks.NewMockTokenRefreshServiceVerifyRefreshClaims(t)
			daoLocationConsent := coremocks.NewMockTokenRefreshDaoLocationConsent(t)
			daoSessionUpsert := coremocks.NewMockTokenRefreshDaoSessionUpsert(t)

			// Expiry is ignored on the access token, since refresh exists to renew an expired one.
			if testCase.verifyClaimsMock != nil {
				verifyClaimsRequest := &servicejsonkeys.VerifyClaimsRequest{
					Usage:       servicejsonkeys.KeyUsageAuth,
					AccessToken: testCase.request.AccessToken,
					Options:     &servicejsonkeys.VerifyClaimsOptions{IgnoreExpired: true},
				}

				serviceVerifyClaims.EXPECT().
					VerifyClaims(mock.Anything, verifyClaimsRequest).
					Return(testCase.verifyClaimsMock.resp, testCase.verifyClaimsMock.err)
			}

			if testCase.verifyRefreshClaimsMock != nil {
				verifyClaimsRequest := &servicejsonkeys.VerifyClaimsRequest{
					Usage:       servicejsonkeys.KeyUsageAuthRefresh,
					AccessToken: testCase.request.RefreshToken,
				}

				serviceVerifyRefreshClaims.EXPECT().
					VerifyClaims(mock.Anything, verifyClaimsRequest).
					Return(testCase.verifyRefreshClaimsMock.resp, testCase.verifyRefreshClaimsMock.err)
			}

			if testCase.revocationMock != nil {
				mockDaoRevocationExist.EXPECT().
					Exec(mock.Anything, &dao.RefreshTokenRevocationExistRequest{JTIs: testCase.revocationMock.jtis}).
					Return(false, nil)
			}

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, &dao.CredentialsSelectRequest{ID: userID}).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			if testCase.signClaimsMock != nil {
				refreshTokenClaims := testCase.verifyRefreshClaimsMock.resp
				accessTokenClaims := &session.Claims{
					UserID:         &userID,
					Roles:          []string{testCase.daoMock.resp.Role},
					RefreshTokenID: refreshTokenClaims.Jti,
					SessionID:      testCase.signClaimsMock.sessionID,
					SessionExp:     refreshTokenClaims.SessionExpiry(),
				}
				claimsSignRequest := &servicejsonkeys.ClaimsSignRequest{
					Usage:   servicejsonkeys.KeyUsageAuth,
					Payload: lo.Must(grpcf.MarshalJSONAsAny(accessTokenClaims)),
				}

				serviceSignClaims.EXPECT().
					ClaimsSign(mock.Anything, claimsSignRequest).
					Return(&servicejsonkeys.ClaimsSignResponse{Token: "new-access-token"}, testCase.signClaimsMock.err)
			}

			// Without consent, the renewal writes the row alone, with no consent read.
			if testCase.consentMock != nil {
				daoLocationConsent.EXPECT().
					Exec(mock.Anything, &dao.CredentialsLocationConsentRequest{ID: userID}).
					Return(testCase.consentMock.resp, testCase.consentMock.err)
			}

			if testCase.sessionUpsert != nil {
				daoSessionUpsert.EXPECT().
					Exec(mock.Anything, matchSessionUpsert(t, testCase.sessionUpsert)).
					Return(nil)
			}

			service := core.NewTokenRefresh(
				mockDao,
				mockDaoRevocationExist,
				daoLocationConsent,
				daoSessionUpsert,
				serviceSignClaims,
				serviceVerifyClaims,
				serviceVerifyRefreshClaims,
				transactiontest.NewTransactor(),
			)

			resp, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			if testCase.expectNotErr != nil {
				require.NotErrorIs(t, err, testCase.expectNotErr)
			}
		})
	}
}
