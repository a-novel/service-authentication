package core_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/transaction/transactiontest"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/lib"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

// TestTokenCreate also covers what every login shares, once: signing the token pair, recording the
// session and bounding the account's sessions. The other operations that open or renew a session
// keep one success path through them.
func TestTokenCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	leeway := 5 * time.Minute

	passwordArgon2ed, err := lib.GenerateArgon2("password")
	require.NoError(t, err)

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	visitor := core.Visitor{UserAgent: "Firefox", IPAddress: "192.0.2.1", Region: "FR", City: "Paris"}
	request := &core.TokenCreateRequest{Email: "user@provider.com", Password: "password", Visitor: visitor}

	// The access token carries the account's role, whichever it is.
	credentials := &dao.Credentials{ID: userID, Password: passwordArgon2ed, Role: authconfig.RoleAdmin}
	consentingCredentials := changed(*credentials, func(credentials *dao.Credentials) {
		credentials.LocationConsent = true
	})

	// The mock refresh token opens its own session, with no expiry. Without consent, the session
	// records the user agent alone.
	sessionWithoutLocation := &dao.SessionUpsertRequest{
		ID:        uuid.MustParse(mockUnsignedJTI),
		UserID:    userID,
		UserAgent: "Firefox",
		ExpiresAt: time.Unix(0, 0),
	}
	sessionWithLocation := changed(*sessionWithoutLocation, func(request *dao.SessionUpsertRequest) {
		request.IPAddress, request.Region, request.City = "192.0.2.1", "FR", "Paris"
	})

	type daoMock struct {
		resp *dao.Credentials
		err  error
	}

	type errMock struct {
		err error
	}

	type consentMock struct {
		resp bool
		err  error
	}

	type sessionMock struct {
		request *dao.SessionUpsertRequest
		err     error
	}

	type testCase struct {
		name string

		request *core.TokenCreateRequest

		daoMock         *daoMock
		signRefreshMock *errMock
		signAccessMock  *errMock
		// consentMock answers the locked consent read of a consenting account, in the transaction that
		// records its session.
		consentMock *consentMock
		sessionMock *sessionMock
		// boundsMock answers both bounds of the account's sessions.
		boundsMock *errMock

		expect    *core.Token
		expectErr error
	}

	success := &testCase{
		name: "Success",

		request: request,

		daoMock:         &daoMock{resp: credentials},
		signRefreshMock: &errMock{},
		signAccessMock:  &errMock{},
		sessionMock:     &sessionMock{request: sessionWithoutLocation},
		boundsMock:      &errMock{},

		expect: &core.Token{AccessToken: "access-token", RefreshToken: mockUnsignedRefreshToken},
	}

	testCases := []*testCase{
		success,
		changed(*success, func(variant *testCase) {
			// An account that consents records its location, after a locked read of the consent.
			variant.name = "Success/LocationConsent"
			variant.daoMock = &daoMock{resp: consentingCredentials}
			variant.consentMock = &consentMock{resp: true}
			variant.sessionMock = &sessionMock{request: sessionWithLocation}
		}),
		changed(*success, func(variant *testCase) {
			// A consent withdrawn since the login read the account records no location: the locked read
			// decides.
			variant.name = "Success/ConsentWithdrawn"
			variant.daoMock = &daoMock{resp: consentingCredentials}
			variant.consentMock = &consentMock{}
		}),
		changed(*success, func(variant *testCase) {
			// Failed bounds leave the login standing.
			variant.name = "Success/BoundsFail"
			variant.boundsMock = &errMock{err: errFoo}
		}),
		changed(*success, func(variant *testCase) {
			// A user agent past the limit keeps its start.
			variant.name = "Success/LongUserAgent"
			variant.request = changed(*request, func(request *core.TokenCreateRequest) {
				request.Visitor.UserAgent = strings.Repeat("é", core.VisitorFieldMaxLength+1)
			})
			variant.sessionMock = &sessionMock{
				request: changed(*sessionWithoutLocation, func(request *dao.SessionUpsertRequest) {
					request.UserAgent = strings.Repeat("é", core.VisitorFieldMaxLength)
				}),
			}
		}),
		{
			name: "Error/InvalidVisitorIP",

			request: changed(*request, func(request *core.TokenCreateRequest) { request.Visitor.IPAddress = "not-an-ip" }),

			expectErr: core.ErrInvalidRequest,
		},
		{
			// An unknown email costs a hash, as a wrong password does.
			name: "Error/UnknownEmail",

			request: request,

			daoMock: &daoMock{err: dao.ErrCredentialsNotFound},

			expectErr: core.ErrCredentialsNotFound,
		},
		{
			name: "Error/SelectCredentials",

			request: request,

			daoMock: &daoMock{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/WrongPassword",

			request: changed(*request, func(request *core.TokenCreateRequest) { request.Password = "fake-password" }),

			daoMock: &daoMock{resp: credentials},

			expectErr: lib.ErrInvalidPassword,
		},
		changed(*success, func(variant *testCase) {
			variant.name = "Error/IssueRefreshToken"
			variant.signRefreshMock = &errMock{err: errFoo}
			variant.signAccessMock, variant.sessionMock, variant.boundsMock = nil, nil, nil
			variant.expect, variant.expectErr = nil, errFoo
		}),
		changed(*success, func(variant *testCase) {
			variant.name = "Error/IssueToken"
			variant.signAccessMock = &errMock{err: errFoo}
			variant.sessionMock, variant.boundsMock = nil, nil
			variant.expect, variant.expectErr = nil, errFoo
		}),
		changed(*success, func(variant *testCase) {
			// A failed record fails the login.
			variant.name = "Error/SessionUpsert"
			variant.sessionMock = &sessionMock{request: sessionWithoutLocation, err: errFoo}
			variant.boundsMock = nil
			variant.expect, variant.expectErr = nil, errFoo
		}),
		changed(*success, func(variant *testCase) {
			variant.name = "Error/ConsentRead"
			variant.daoMock = &daoMock{resp: consentingCredentials}
			variant.consentMock = &consentMock{err: errFoo}
			variant.sessionMock, variant.boundsMock = nil, nil
			variant.expect, variant.expectErr = nil, errFoo
		}),
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockDao := coremocks.NewMockTokenCreateDao(t)
			daoLocationConsent := coremocks.NewMockTokenCreateDaoLocationConsent(t)
			daoSessionUpsert := coremocks.NewMockTokenCreateDaoSessionUpsert(t)
			daoSessionRevokeOldest := coremocks.NewMockTokenCreateDaoSessionRevokeOldest(t)
			daoSessionTrim := coremocks.NewMockTokenCreateDaoSessionTrim(t)
			serviceSignClaims := coremocks.NewMockTokenCreateServiceSignClaims(t)
			transactor := transactiontest.NewTransactor()

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, &dao.CredentialsSelectByEmailRequest{Email: testCase.request.Email}).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			if testCase.signRefreshMock != nil {
				claimsSignRequest := &servicejsonkeys.ClaimsSignRequest{
					Usage:   servicejsonkeys.KeyUsageAuthRefresh,
					Payload: lo.Must(grpcf.MarshalJSONAsAny(session.RefreshClaimsForm{UserID: userID})),
				}

				serviceSignClaims.EXPECT().
					ClaimsSign(mock.Anything, claimsSignRequest).
					Return(&servicejsonkeys.ClaimsSignResponse{Token: mockUnsignedRefreshToken}, testCase.signRefreshMock.err)
			}

			// The access token holds the refresh token's JTI, which binds the pair, and its session.
			if testCase.signAccessMock != nil {
				accessTokenClaims := session.Claims{
					UserID:         &userID,
					Roles:          []string{credentials.Role},
					RefreshTokenID: mockUnsignedJTI,
					SessionID:      mockUnsignedJTI,
				}
				claimsSignRequest := &servicejsonkeys.ClaimsSignRequest{
					Usage:   servicejsonkeys.KeyUsageAuth,
					Payload: lo.Must(grpcf.MarshalJSONAsAny(accessTokenClaims)),
				}

				serviceSignClaims.EXPECT().
					ClaimsSign(mock.Anything, claimsSignRequest).
					Return(&servicejsonkeys.ClaimsSignResponse{Token: "access-token"}, testCase.signAccessMock.err)
			}

			// Without consent, the session is written alone, with no transaction nor consent read.
			var expectTransactions int

			if testCase.consentMock != nil {
				expectTransactions = 1

				daoLocationConsent.EXPECT().
					Exec(mock.Anything, &dao.CredentialsLocationConsentRequest{ID: userID}).
					Return(testCase.consentMock.resp, testCase.consentMock.err)
			}

			if testCase.sessionMock != nil {
				daoSessionUpsert.EXPECT().
					Exec(mock.Anything, matchSessionUpsert(t, testCase.sessionMock.request)).
					Return(testCase.sessionMock.err)
			}

			// The login keeps the account's 100 active sessions active last, and as many logged-out ones.
			if testCase.boundsMock != nil {
				daoSessionRevokeOldest.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(request *dao.SessionRevokeOldestRequest) bool {
						return request.UserID == userID && request.Keep == core.SessionListLimit &&
							request.Limit == core.SessionTrimBatch && request.Leeway == leeway &&
							time.Since(request.Now) < time.Minute
					})).
					Return(testCase.boundsMock.err)
				daoSessionTrim.EXPECT().
					Exec(mock.Anything, &dao.SessionTrimRequest{
						UserID: userID, Keep: core.SessionLoggedOutKept, Limit: core.SessionTrimBatch,
					}).
					Return(testCase.boundsMock.err)
			}

			service := core.NewTokenCreate(
				mockDao, daoLocationConsent, daoSessionUpsert, daoSessionRevokeOldest, daoSessionTrim, serviceSignClaims,
				transactor, leeway,
			)

			resp, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)
			require.Equal(t, expectTransactions, transactor.Calls())
		})
	}
}
