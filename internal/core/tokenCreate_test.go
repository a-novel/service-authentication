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

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/lib"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

func TestTokenCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	passwordRaw := "password"
	passwordArgon2ed, err := lib.GenerateArgon2(passwordRaw, lib.Argon2ParamsDefault)
	require.NoError(t, err)

	type daoMock struct {
		resp *dao.Credentials
		err  error
	}

	type issueTokenMock struct {
		resp *servicejsonkeys.ClaimsSignResponse
		err  error
	}

	type issueRefreshTokenMock struct {
		err error
	}

	// sessionUpsertMock records the session. Under the consent the login read, it does so in a
	// transaction, after a locked consent read that answers consent. The bounds follow a recorded
	// session, and fail with boundsErr.
	type sessionUpsertMock struct {
		consent   bool
		request   *dao.SessionUpsertRequest
		err       error
		boundsErr error
	}

	leeway := 5 * time.Minute

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	visitor := core.Visitor{UserAgent: "Firefox", IPAddress: "192.0.2.1", Region: "FR", City: "Paris"}

	// The mock refresh token opens its own session, with no expiry.
	sessionWithoutLocation := &dao.SessionUpsertRequest{
		ID:        uuid.MustParse(mockUnsignedJTI),
		UserID:    userID,
		UserAgent: "Firefox",
		ExpiresAt: time.Unix(0, 0),
	}

	testCases := []struct {
		name string

		request *core.TokenCreateRequest

		daoMock               *daoMock
		issueRefreshTokenMock *issueRefreshTokenMock
		issueTokenMock        *issueTokenMock
		sessionUpsertMock     *sessionUpsertMock

		expect    *core.Token
		expectErr error
	}{
		{
			name: "Success",

			request: &core.TokenCreateRequest{Email: "user@provider.com", Password: passwordRaw, Visitor: visitor},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password: passwordArgon2ed,
					Role:     authconfig.RoleUser,
				},
			},

			issueRefreshTokenMock: &issueRefreshTokenMock{},

			issueTokenMock: &issueTokenMock{
				resp: &servicejsonkeys.ClaimsSignResponse{
					Token: "access-token",
				},
			},

			// The account does not consent to locations, so the session records the user agent alone.
			sessionUpsertMock: &sessionUpsertMock{request: sessionWithoutLocation},

			expect: &core.Token{
				AccessToken:  "access-token",
				RefreshToken: mockUnsignedRefreshToken,
			},
		},
		{
			name: "Success/RoleUser",

			request: &core.TokenCreateRequest{Email: "user@provider.com", Password: passwordRaw, Visitor: visitor},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password: passwordArgon2ed,
					Role:     authconfig.RoleUser,
				},
			},

			issueRefreshTokenMock: &issueRefreshTokenMock{},

			issueTokenMock: &issueTokenMock{
				resp: &servicejsonkeys.ClaimsSignResponse{
					Token: "access-token",
				},
			},

			// The account does not consent to locations, so the session records the user agent alone.
			sessionUpsertMock: &sessionUpsertMock{request: sessionWithoutLocation},

			expect: &core.Token{
				AccessToken:  "access-token",
				RefreshToken: mockUnsignedRefreshToken,
			},
		},
		{
			name: "Success/RoleAdmin",

			request: &core.TokenCreateRequest{Email: "user@provider.com", Password: passwordRaw, Visitor: visitor},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password: passwordArgon2ed,
					Role:     authconfig.RoleAdmin,
				},
			},

			issueRefreshTokenMock: &issueRefreshTokenMock{},

			issueTokenMock: &issueTokenMock{
				resp: &servicejsonkeys.ClaimsSignResponse{
					Token: "access-token",
				},
			},

			// The account does not consent to locations, so the session records the user agent alone.
			sessionUpsertMock: &sessionUpsertMock{request: sessionWithoutLocation},

			expect: &core.Token{
				AccessToken:  "access-token",
				RefreshToken: mockUnsignedRefreshToken,
			},
		},
		{
			name: "Success/RoleSuperAdmin",

			request: &core.TokenCreateRequest{Email: "user@provider.com", Password: passwordRaw, Visitor: visitor},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password: passwordArgon2ed,
					Role:     authconfig.RoleSuperAdmin,
				},
			},

			issueRefreshTokenMock: &issueRefreshTokenMock{},

			issueTokenMock: &issueTokenMock{
				resp: &servicejsonkeys.ClaimsSignResponse{
					Token: "access-token",
				},
			},

			// The account does not consent to locations, so the session records the user agent alone.
			sessionUpsertMock: &sessionUpsertMock{request: sessionWithoutLocation},

			expect: &core.Token{
				AccessToken:  "access-token",
				RefreshToken: mockUnsignedRefreshToken,
			},
		},
		{
			// An account that consents records its location, after a locked read of the consent.
			name: "Success/LocationConsent",

			request: &core.TokenCreateRequest{Email: "user@provider.com", Password: passwordRaw, Visitor: visitor},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID:              uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password:        passwordArgon2ed,
					Role:            authconfig.RoleUser,
					LocationConsent: true,
				},
			},

			issueRefreshTokenMock: &issueRefreshTokenMock{},

			issueTokenMock: &issueTokenMock{
				resp: &servicejsonkeys.ClaimsSignResponse{
					Token: "access-token",
				},
			},

			sessionUpsertMock: &sessionUpsertMock{
				consent: true,
				request: changed(*sessionWithoutLocation, func(request *dao.SessionUpsertRequest) {
					request.IPAddress = "192.0.2.1"
					request.Region = "FR"
					request.City = "Paris"
				}),
			},

			expect: &core.Token{
				AccessToken:  "access-token",
				RefreshToken: mockUnsignedRefreshToken,
			},
		},
		{
			// A consent withdrawn since the login read the account records no location: the locked read
			// decides.
			name: "Success/ConsentWithdrawn",

			request: &core.TokenCreateRequest{Email: "user@provider.com", Password: passwordRaw, Visitor: visitor},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID:              uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password:        passwordArgon2ed,
					Role:            authconfig.RoleUser,
					LocationConsent: true,
				},
			},

			issueRefreshTokenMock: &issueRefreshTokenMock{},

			issueTokenMock: &issueTokenMock{
				resp: &servicejsonkeys.ClaimsSignResponse{
					Token: "access-token",
				},
			},

			sessionUpsertMock: &sessionUpsertMock{request: sessionWithoutLocation},

			expect: &core.Token{
				AccessToken:  "access-token",
				RefreshToken: mockUnsignedRefreshToken,
			},
		},
		{
			// Failed bounds leave the login standing.
			name: "Success/BoundsFail",

			request: &core.TokenCreateRequest{Email: "user@provider.com", Password: passwordRaw, Visitor: visitor},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password: passwordArgon2ed,
					Role:     authconfig.RoleUser,
				},
			},

			issueRefreshTokenMock: &issueRefreshTokenMock{},

			issueTokenMock: &issueTokenMock{
				resp: &servicejsonkeys.ClaimsSignResponse{
					Token: "access-token",
				},
			},

			sessionUpsertMock: &sessionUpsertMock{request: sessionWithoutLocation, boundsErr: errFoo},

			expect: &core.Token{
				AccessToken:  "access-token",
				RefreshToken: mockUnsignedRefreshToken,
			},
		},
		{
			// A user agent past the limit keeps its start.
			name: "Success/LongUserAgent",

			request: &core.TokenCreateRequest{
				Email:    "user@provider.com",
				Password: passwordRaw,
				Visitor:  core.Visitor{UserAgent: strings.Repeat("é", core.VisitorFieldMaxLength+1)},
			},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password: passwordArgon2ed,
					Role:     authconfig.RoleUser,
				},
			},

			issueRefreshTokenMock: &issueRefreshTokenMock{},

			issueTokenMock: &issueTokenMock{
				resp: &servicejsonkeys.ClaimsSignResponse{
					Token: "access-token",
				},
			},

			sessionUpsertMock: &sessionUpsertMock{
				request: changed(*sessionWithoutLocation, func(request *dao.SessionUpsertRequest) {
					request.UserAgent = strings.Repeat("é", core.VisitorFieldMaxLength)
				}),
			},

			expect: &core.Token{
				AccessToken:  "access-token",
				RefreshToken: mockUnsignedRefreshToken,
			},
		},
		{
			// A failed record fails the login.
			name: "Error/SessionUpsert",

			request: &core.TokenCreateRequest{Email: "user@provider.com", Password: passwordRaw, Visitor: visitor},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password: passwordArgon2ed,
					Role:     authconfig.RoleUser,
				},
			},

			issueRefreshTokenMock: &issueRefreshTokenMock{},

			issueTokenMock: &issueTokenMock{
				resp: &servicejsonkeys.ClaimsSignResponse{
					Token: "access-token",
				},
			},

			sessionUpsertMock: &sessionUpsertMock{request: sessionWithoutLocation, err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/InvalidVisitorIP",

			request: &core.TokenCreateRequest{
				Email:    "user@provider.com",
				Password: passwordRaw,
				Visitor:  core.Visitor{IPAddress: "not-an-ip"},
			},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/WrongPassword",

			request: &core.TokenCreateRequest{Email: "user@provider.com", Password: "fake-password"},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password: passwordArgon2ed,
					Role:     authconfig.RoleUser,
				},
			},

			expectErr: lib.ErrInvalidPassword,
		},
		{
			name: "Error/IssueToken",

			request: &core.TokenCreateRequest{Email: "user@provider.com", Password: passwordRaw, Visitor: visitor},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password: passwordArgon2ed,
					Role:     authconfig.RoleUser,
				},
			},

			issueRefreshTokenMock: &issueRefreshTokenMock{},

			issueTokenMock: &issueTokenMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/IssueRefreshToken",

			request: &core.TokenCreateRequest{Email: "user@provider.com", Password: passwordRaw, Visitor: visitor},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password: passwordArgon2ed,
					Role:     authconfig.RoleUser,
				},
			},

			issueRefreshTokenMock: &issueRefreshTokenMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/SelectCredentials",

			request: &core.TokenCreateRequest{Email: "user@provider.com", Password: "fake-password"},

			daoMock: &daoMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			mockDao := coremocks.NewMockTokenCreateDao(t)
			daoLocationConsent := coremocks.NewMockTokenCreateDaoLocationConsent(t)
			daoSessionUpsert := coremocks.NewMockTokenCreateDaoSessionUpsert(t)
			daoSessionRevokeOldest := coremocks.NewMockTokenCreateDaoSessionRevokeOldest(t)
			daoSessionTrim := coremocks.NewMockTokenCreateDaoSessionTrim(t)
			serviceSignClaims := coremocks.NewMockTokenCreateServiceSignClaims(t)
			transactor := coremocks.NewMockTransactor(t)

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, &dao.CredentialsSelectByEmailRequest{
						Email: testCase.request.Email,
					}).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			if testCase.issueRefreshTokenMock != nil {
				claimsSignRequest := &servicejsonkeys.ClaimsSignRequest{
					Usage: servicejsonkeys.KeyUsageAuthRefresh,
					Payload: lo.Must(grpcf.MarshalJSONAsAny(session.RefreshClaimsForm{
						UserID: testCase.daoMock.resp.ID,
					})),
				}

				serviceSignClaims.EXPECT().
					ClaimsSign(mock.Anything, claimsSignRequest).
					Return(
						&servicejsonkeys.ClaimsSignResponse{
							Token: mockUnsignedRefreshToken,
						},
						testCase.issueRefreshTokenMock.err,
					)
			}

			if testCase.issueTokenMock != nil {
				accessTokenClaims := session.Claims{
					UserID:         &testCase.daoMock.resp.ID,
					Roles:          []string{testCase.daoMock.resp.Role},
					RefreshTokenID: mockUnsignedJTI,
					SessionID:      mockUnsignedJTI,
				}
				claimsSignRequest := &servicejsonkeys.ClaimsSignRequest{
					Usage:   servicejsonkeys.KeyUsageAuth,
					Payload: lo.Must(grpcf.MarshalJSONAsAny(accessTokenClaims)),
				}

				serviceSignClaims.EXPECT().
					ClaimsSign(mock.Anything, claimsSignRequest).
					Return(testCase.issueTokenMock.resp, testCase.issueTokenMock.err)
			}

			// Without consent, the session is written alone, with no transaction nor consent read.
			if testCase.sessionUpsertMock != nil && testCase.daoMock.resp.LocationConsent {
				transactor.EXPECT().WithinTx(mock.Anything, mock.Anything).RunAndReturn(passThroughTx).Once()
				daoLocationConsent.EXPECT().
					Exec(mock.Anything, &dao.CredentialsLocationConsentRequest{ID: userID}).
					Return(testCase.sessionUpsertMock.consent, nil)
			}

			if testCase.sessionUpsertMock != nil {
				daoSessionUpsert.EXPECT().
					Exec(mock.Anything, matchSessionUpsert(t, testCase.sessionUpsertMock.request)).
					Return(testCase.sessionUpsertMock.err)
			}

			// The login keeps the account's 100 active sessions active last, and as many logged-out ones.
			if testCase.sessionUpsertMock != nil && testCase.sessionUpsertMock.err == nil {
				daoSessionRevokeOldest.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(request *dao.SessionRevokeOldestRequest) bool {
						return request.UserID == userID && request.Keep == core.SessionListLimit &&
							request.Limit == core.SessionTrimBatch && request.Leeway == leeway &&
							time.Since(request.Now) < time.Minute
					})).
					Return(testCase.sessionUpsertMock.boundsErr)
				daoSessionTrim.EXPECT().
					Exec(mock.Anything, &dao.SessionTrimRequest{
						UserID: userID, Keep: core.SessionLoggedOutKept, Limit: core.SessionTrimBatch,
					}).
					Return(testCase.sessionUpsertMock.boundsErr)
			}

			service := core.NewTokenCreate(
				mockDao, daoLocationConsent, daoSessionUpsert, daoSessionRevokeOldest, daoSessionTrim, serviceSignClaims,
				transactor, leeway,
			)

			resp, err := service.Exec(ctx, testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			mockDao.AssertExpectations(t)
			daoLocationConsent.AssertExpectations(t)
			daoSessionUpsert.AssertExpectations(t)
			daoSessionRevokeOldest.AssertExpectations(t)
			daoSessionTrim.AssertExpectations(t)
			serviceSignClaims.AssertExpectations(t)
			transactor.AssertExpectations(t)
		})
	}
}
