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

	type sessionUpsertMock struct {
		request *dao.SessionUpsertRequest
		err     error
	}

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
			daoSessionUpsert := coremocks.NewMockTokenCreateDaoSessionUpsert(t)
			serviceSignClaims := coremocks.NewMockTokenCreateServiceSignClaims(t)

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

			if testCase.sessionUpsertMock != nil {
				daoSessionUpsert.EXPECT().
					Exec(mock.Anything, matchSessionUpsert(t, testCase.sessionUpsertMock.request)).
					Return(testCase.sessionUpsertMock.err)
			}

			service := core.NewTokenCreate(mockDao, daoSessionUpsert, serviceSignClaims)

			resp, err := service.Exec(ctx, testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			mockDao.AssertExpectations(t)
			daoSessionUpsert.AssertExpectations(t)
			serviceSignClaims.AssertExpectations(t)
		})
	}
}
