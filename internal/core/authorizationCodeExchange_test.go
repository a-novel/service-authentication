package core_test

import (
	"crypto/sha256"
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

	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

func TestAuthorizationCodeExchange(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	leeway := 5 * time.Minute
	now := time.Now().Round(time.Second)
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	secret := strings.Repeat("c", 43)
	codeID := sha256.Sum256([]byte(secret))
	// The verifier of RFC 7636's appendix B, and its S256 challenge.
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

	validRequest := core.AuthorizationCodeExchangeRequest{
		Caller:       studioCaller,
		Code:         secret,
		CodeVerifier: verifier,
		RedirectURI:  studioRedirectURI,
	}

	validCode := dao.AuthorizationCode{
		ID:               codeID[:],
		UserID:           userID,
		ClientID:         "studio",
		RedirectURI:      studioRedirectURI,
		CodeChallenge:    challenge,
		SessionID:        "session-id",
		SessionExpiresAt: now.Add(time.Hour),
		ExpiresAt:        now.Add(time.Minute),
	}

	credentials := &dao.Credentials{ID: userID, Role: "auth:user"}

	type daoSelectMock struct {
		resp *dao.AuthorizationCode
		err  error
	}

	type errMock struct {
		err error
	}

	type daoRevocationExistMock struct {
		resp bool
		err  error
	}

	type daoCredentialsMock struct {
		resp *dao.Credentials
		err  error
	}

	testCases := []struct {
		name string

		request *core.AuthorizationCodeExchangeRequest

		daoSelectMock           *daoSelectMock
		daoRevocationExistMock  *daoRevocationExistMock
		daoCredentialsMock      *daoCredentialsMock
		signMock                *errMock
		daoConsumeMock          *errMock
		daoRevocationInsertMock *errMock

		expectErr error
		// expectNotErr is a sentinel the returned error must not carry.
		expectNotErr error
	}{
		{
			name: "Success",

			request: &validRequest,

			daoSelectMock:          &daoSelectMock{resp: &validCode},
			daoRevocationExistMock: &daoRevocationExistMock{},
			daoCredentialsMock:     &daoCredentialsMock{resp: credentials},
			signMock:               &errMock{},
			daoConsumeMock:         &errMock{},
		},
		{
			// A server that verifies no service token sees no caller, and reads nothing.
			name: "Error/NoCaller",

			request: changed(validRequest, func(request *core.AuthorizationCodeExchangeRequest) { request.Caller = "" }),

			expectErr: core.ErrAuthorizationCodeExchangeForbiddenCaller,
		},
		{
			name: "Error/CallerOutsideRegistry",

			request: changed(validRequest, func(request *core.AuthorizationCodeExchangeRequest) {
				request.Caller = accountPlatformCaller
			}),

			expectErr: core.ErrAuthorizationCodeExchangeForbiddenCaller,
		},
		{
			name: "Error/VerifierTooShort",

			request: changed(validRequest, func(request *core.AuthorizationCodeExchangeRequest) {
				request.CodeVerifier = verifier[:42]
			}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/UnknownCode",

			request: &validRequest,

			daoSelectMock: &daoSelectMock{err: dao.ErrAuthorizationCodeSelectNotFound},

			expectErr: core.ErrAuthorizationCodeExchangeUnknownCode,
		},
		{
			name: "Error/Select",

			request: &validRequest,

			daoSelectMock: &daoSelectMock{err: errFoo},

			expectErr: errFoo,
		},
		{
			// A second exchange ends the session the first one extended to the client.
			name: "Error/Reused",

			request: &validRequest,

			daoSelectMock: &daoSelectMock{resp: changed(validCode, func(code *dao.AuthorizationCode) {
				code.ConsumedAt = lo.ToPtr(now.Add(-time.Second))
			})},
			daoRevocationInsertMock: &errMock{},

			expectErr: core.ErrAuthorizationCodeExchangeReused,
		},
		{
			// Reuse is checked first: a replay from another client also ends the session.
			name: "Error/ReusedByOtherClient",

			request: changed(validRequest, func(request *core.AuthorizationCodeExchangeRequest) {
				request.Caller = "system:serviceaccount:forum:web"
			}),

			daoSelectMock: &daoSelectMock{resp: changed(validCode, func(code *dao.AuthorizationCode) {
				code.ConsumedAt = lo.ToPtr(now.Add(-time.Second))
			})},
			daoRevocationInsertMock: &errMock{},

			expectErr: core.ErrAuthorizationCodeExchangeReused,
		},
		{
			name: "Error/ReusedRevocation",

			request: &validRequest,

			daoSelectMock: &daoSelectMock{resp: changed(validCode, func(code *dao.AuthorizationCode) {
				code.ConsumedAt = lo.ToPtr(now.Add(-time.Second))
			})},
			daoRevocationInsertMock: &errMock{err: errFoo},

			expectErr:    errFoo,
			expectNotErr: core.ErrAuthorizationCodeExchangeReused,
		},
		{
			// Another exchange consumed the code between its read and its consumption.
			name: "Error/ReusedConcurrently",

			request: &validRequest,

			daoSelectMock:           &daoSelectMock{resp: &validCode},
			daoRevocationExistMock:  &daoRevocationExistMock{},
			daoCredentialsMock:      &daoCredentialsMock{resp: credentials},
			signMock:                &errMock{},
			daoConsumeMock:          &errMock{err: dao.ErrAuthorizationCodeConsumeNotFound},
			daoRevocationInsertMock: &errMock{},

			expectErr: core.ErrAuthorizationCodeExchangeReused,
		},
		{
			name: "Error/Expired",

			request: &validRequest,

			daoSelectMock: &daoSelectMock{resp: changed(validCode, func(code *dao.AuthorizationCode) {
				code.ExpiresAt = now.Add(-time.Second)
			})},

			expectErr: core.ErrAuthorizationCodeExchangeExpired,
		},
		{
			// A code issued to forum, presented by studio.
			name: "Error/ForeignCode",

			request: &validRequest,

			daoSelectMock: &daoSelectMock{resp: changed(validCode, func(code *dao.AuthorizationCode) {
				code.ClientID = "forum"
				code.RedirectURI = forumRedirectURI
			})},

			expectErr: core.ErrAuthorizationCodeExchangeForeignCode,
		},
		{
			name: "Error/RedirectURIQuery",

			request: changed(validRequest, func(request *core.AuthorizationCodeExchangeRequest) {
				request.RedirectURI = studioRedirectURI + "?next=/"
			}),

			daoSelectMock: &daoSelectMock{resp: &validCode},

			expectErr: core.ErrAuthorizationCodeExchangeRedirectURIMismatch,
		},
		{
			name: "Error/RedirectURIPrefix",

			request: changed(validRequest, func(request *core.AuthorizationCodeExchangeRequest) {
				request.RedirectURI = "https://studio.example.com/auth"
			}),

			daoSelectMock: &daoSelectMock{resp: &validCode},

			expectErr: core.ErrAuthorizationCodeExchangeRedirectURIMismatch,
		},
		{
			name: "Error/WrongVerifier",

			request: changed(validRequest, func(request *core.AuthorizationCodeExchangeRequest) {
				request.CodeVerifier = strings.Repeat("v", 43)
			}),

			daoSelectMock: &daoSelectMock{resp: &validCode},

			expectErr: core.ErrAuthorizationCodeExchangeInvalidVerifier,
		},
		{
			// The plain method sends the challenge as the verifier, which S256 refuses.
			name: "Error/PlainVerifier",

			request: changed(validRequest, func(request *core.AuthorizationCodeExchangeRequest) {
				request.CodeVerifier = challenge
			}),

			daoSelectMock: &daoSelectMock{resp: &validCode},

			expectErr: core.ErrAuthorizationCodeExchangeInvalidVerifier,
		},
		{
			name: "Error/SessionExpired",

			request: &validRequest,

			daoSelectMock: &daoSelectMock{resp: changed(validCode, func(code *dao.AuthorizationCode) {
				code.SessionExpiresAt = now.Add(-time.Second)
			})},

			expectErr: core.ErrAuthorizationCodeExchangeSessionEnded,
		},
		{
			// The person signed out on another platform after the code was issued.
			name: "Error/SessionRevoked",

			request: &validRequest,

			daoSelectMock:          &daoSelectMock{resp: &validCode},
			daoRevocationExistMock: &daoRevocationExistMock{resp: true},

			expectErr: core.ErrAuthorizationCodeExchangeSessionEnded,
		},
		{
			name: "Error/RevocationExist",

			request: &validRequest,

			daoSelectMock:          &daoSelectMock{resp: &validCode},
			daoRevocationExistMock: &daoRevocationExistMock{err: errFoo},

			expectErr: errFoo,
		},
		{
			// An exchange signs in an existing account, and never creates one.
			name: "Error/UnknownAccount",

			request: &validRequest,

			daoSelectMock:          &daoSelectMock{resp: &validCode},
			daoRevocationExistMock: &daoRevocationExistMock{},
			daoCredentialsMock:     &daoCredentialsMock{err: dao.ErrCredentialsSelectNotFound},

			expectErr: core.ErrAuthorizationCodeExchangeUnknownCode,
		},
		{
			// A failed signature leaves the code unconsumed, so the client may retry.
			name: "Error/Sign",

			request: &validRequest,

			daoSelectMock:          &daoSelectMock{resp: &validCode},
			daoRevocationExistMock: &daoRevocationExistMock{},
			daoCredentialsMock:     &daoCredentialsMock{resp: credentials},
			signMock:               &errMock{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Consume",

			request: &validRequest,

			daoSelectMock:          &daoSelectMock{resp: &validCode},
			daoRevocationExistMock: &daoRevocationExistMock{},
			daoCredentialsMock:     &daoCredentialsMock{resp: credentials},
			signMock:               &errMock{},
			daoConsumeMock:         &errMock{err: errFoo},

			expectErr:    errFoo,
			expectNotErr: core.ErrAuthorizationCodeExchangeReused,
		},
	}

	// The exchanged pair joins the code's session, and its refresh token stops with that session.
	refreshTokenForm := session.RefreshClaimsForm{
		UserID:     userID,
		SessionID:  validCode.SessionID,
		SessionExp: validCode.SessionExpiresAt.Unix(),
	}
	refreshToken := mockRefreshToken(session.RefreshClaims{
		Jti:        "refresh-token-id",
		Exp:        now.Add(7 * 24 * time.Hour).Unix(),
		UserID:     userID,
		SessionID:  refreshTokenForm.SessionID,
		SessionExp: refreshTokenForm.SessionExp,
	})
	accessTokenClaims := session.Claims{
		UserID:         &userID,
		Roles:          []string{credentials.Role},
		RefreshTokenID: "refresh-token-id",
		SessionID:      refreshTokenForm.SessionID,
		SessionExp:     refreshTokenForm.SessionExp,
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			daoSelect := coremocks.NewMockAuthorizationCodeExchangeDaoSelect(t)
			daoConsume := coremocks.NewMockAuthorizationCodeExchangeDaoConsume(t)
			daoCredentials := coremocks.NewMockAuthorizationCodeExchangeDaoCredentials(t)
			daoRevocationExist := coremocks.NewMockAuthorizationCodeExchangeDaoRevocationExist(t)
			daoRevocationInsert := coremocks.NewMockAuthorizationCodeExchangeDaoRevocationInsert(t)
			serviceSignClaims := coremocks.NewMockAuthorizationCodeExchangeServiceSignClaims(t)

			if testCase.daoSelectMock != nil {
				daoSelect.EXPECT().
					Exec(mock.Anything, &dao.AuthorizationCodeSelectRequest{ID: codeID[:]}).
					Return(testCase.daoSelectMock.resp, testCase.daoSelectMock.err)
			}

			if testCase.daoRevocationExistMock != nil {
				daoRevocationExist.EXPECT().
					Exec(mock.Anything, &dao.RefreshTokenRevocationExistRequest{JTIs: []string{"session-id"}}).
					Return(testCase.daoRevocationExistMock.resp, testCase.daoRevocationExistMock.err)
			}

			if testCase.daoCredentialsMock != nil {
				daoCredentials.EXPECT().
					Exec(mock.Anything, &dao.CredentialsSelectRequest{ID: userID}).
					Return(testCase.daoCredentialsMock.resp, testCase.daoCredentialsMock.err)
			}

			if testCase.signMock != nil {
				serviceSignClaims.EXPECT().
					ClaimsSign(mock.Anything, &servicejsonkeys.ClaimsSignRequest{
						Usage:   servicejsonkeys.KeyUsageAuthRefresh,
						Payload: lo.Must(grpcf.MarshalJSONAsAny(refreshTokenForm)),
					}).
					Return(&servicejsonkeys.ClaimsSignResponse{Token: refreshToken}, testCase.signMock.err)
			}

			if testCase.signMock != nil && testCase.signMock.err == nil {
				serviceSignClaims.EXPECT().
					ClaimsSign(mock.Anything, &servicejsonkeys.ClaimsSignRequest{
						Usage:   servicejsonkeys.KeyUsageAuth,
						Payload: lo.Must(grpcf.MarshalJSONAsAny(accessTokenClaims)),
					}).
					Return(&servicejsonkeys.ClaimsSignResponse{Token: "access-token"}, nil)
			}

			if testCase.daoConsumeMock != nil {
				daoConsume.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(request *dao.AuthorizationCodeConsumeRequest) bool {
						return string(request.ID) == string(codeID[:]) &&
							time.Since(request.Now) < time.Minute
					})).
					Return(&validCode, testCase.daoConsumeMock.err)
			}

			if testCase.daoRevocationInsertMock != nil {
				daoRevocationInsert.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(request *dao.RefreshTokenRevocationInsertRequest) bool {
						return len(request.JTIs) == 1 && request.JTIs[0] == "session-id" &&
							request.ExpiresAt.Equal(validCode.SessionExpiresAt.Add(leeway))
					})).
					Return(testCase.daoRevocationInsertMock.err)
			}

			service := core.NewAuthorizationCodeExchange(
				daoSelect, daoConsume, daoCredentials, daoRevocationExist, daoRevocationInsert, serviceSignClaims,
				oauthPreset, leeway,
			)

			res, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)

			if testCase.expectNotErr != nil {
				require.NotErrorIs(t, err, testCase.expectNotErr)
			}

			if testCase.expectErr == nil {
				require.Equal(t, &core.Token{AccessToken: "access-token", RefreshToken: refreshToken}, res)
			} else {
				require.Nil(t, res)
			}

			daoSelect.AssertExpectations(t)
			daoConsume.AssertExpectations(t)
			daoCredentials.AssertExpectations(t)
			daoRevocationExist.AssertExpectations(t)
			daoRevocationInsert.AssertExpectations(t)
			serviceSignClaims.AssertExpectations(t)
		})
	}
}
