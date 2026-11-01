package core_test

import (
	"crypto/sha256"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/jwt/v2/jws"

	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

func TestTokenGrant(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	leeway := 5 * time.Minute
	accessTokenTTL := time.Hour
	now := time.Now().Round(time.Second)
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	codeID := sha256.Sum256([]byte(authorizationCodeSecret))

	validCode := dao.AuthorizationCode{
		ID:                   codeID[:],
		UserID:               userID,
		ClientID:             "studio",
		RedirectURI:          studioRedirectURI,
		RedirectURIInRequest: true,
		CodeChallenge:        rfc7636Challenge,
		CodeChallengeMethod:  core.CodeChallengeMethodS256,
		SessionID:            "session-id",
		SessionExpiresAt:     now.Add(time.Hour),
		ExpiresAt:            now.Add(time.Minute),
	}
	consumedCode := changed(validCode, func(code *dao.AuthorizationCode) {
		code.ConsumedAt = lo.ToPtr(now.Add(-time.Second))
	})

	credentials := &dao.Credentials{ID: userID, Role: "auth:user"}

	// The code grant's pair joins the code's session, bound to studio, and stops with that session.
	refreshTokenForm := session.RefreshClaimsForm{
		UserID:     userID,
		SessionID:  validCode.SessionID,
		SessionExp: validCode.SessionExpiresAt.Unix(),
		ClientID:   "studio",
	}
	refreshTokenClaims := session.RefreshClaims{
		Jti:        "refresh-token-id",
		Exp:        now.Add(7 * 24 * time.Hour).Unix(),
		UserID:     userID,
		SessionID:  refreshTokenForm.SessionID,
		SessionExp: refreshTokenForm.SessionExp,
		ClientID:   "studio",
	}
	refreshToken := mockRefreshToken(refreshTokenClaims)
	accessTokenClaims := session.Claims{
		UserID:         &userID,
		Roles:          []string{credentials.Role},
		RefreshTokenID: refreshTokenClaims.Jti,
		SessionID:      refreshTokenClaims.SessionID,
		SessionExp:     refreshTokenClaims.SessionExp,
	}

	type respMock[T any] struct {
		resp T
		err  error
	}

	type errMock struct {
		err error
	}

	testCases := []struct {
		name string

		caller string
		params url.Values

		daoSelectMock           *respMock[*dao.AuthorizationCode]
		verifyRefreshMock       *respMock[*session.RefreshClaims]
		daoRevocationExistMock  *respMock[bool]
		daoCredentialsMock      *respMock[*dao.Credentials]
		signRefreshMock         *errMock
		signAccessMock          *errMock
		daoConsumeMock          *errMock
		daoRevocationInsertMock *errMock

		// expectRevocationJTIs are the JTIs the revocation check looks up.
		expectRevocationJTIs []string
		expect               *core.TokenGrantResult
		expectErr            error
		// expectNotErr is a sentinel the returned error must not carry.
		expectNotErr error
	}{
		{
			name: "Success/AuthorizationCode",

			caller: studioCaller,
			params: codeGrantParams(nil),

			daoSelectMock:          &respMock[*dao.AuthorizationCode]{resp: &validCode},
			daoRevocationExistMock: &respMock[bool]{},
			daoCredentialsMock:     &respMock[*dao.Credentials]{resp: credentials},
			signRefreshMock:        &errMock{},
			signAccessMock:         &errMock{},
			daoConsumeMock:         &errMock{},

			expectRevocationJTIs: []string{"session-id"},
			expect: &core.TokenGrantResult{
				AccessToken: "access-token", ExpiresIn: accessTokenTTL, RefreshToken: refreshToken,
			},
		},
		{
			// A client may name itself (RFC 6749 §3.2.1), and the server ignores a parameter it does
			// not know (RFC 6749 §3.2).
			name: "Success/AuthorizationCode/ClientIDAndUnknownParameter",

			caller: studioCaller,
			params: codeGrantParams(func(params url.Values) {
				params.Set("client_id", "studio")
				params["resource"] = []string{"a", "b"}
			}),

			daoSelectMock:          &respMock[*dao.AuthorizationCode]{resp: &validCode},
			daoRevocationExistMock: &respMock[bool]{},
			daoCredentialsMock:     &respMock[*dao.Credentials]{resp: credentials},
			signRefreshMock:        &errMock{},
			signAccessMock:         &errMock{},
			daoConsumeMock:         &errMock{},

			expectRevocationJTIs: []string{"session-id"},
			expect: &core.TokenGrantResult{
				AccessToken: "access-token", ExpiresIn: accessTokenTTL, RefreshToken: refreshToken,
			},
		},
		{
			// An authorization request without redirect_uri lets the token request omit it.
			name: "Success/AuthorizationCode/NoRedirectURI",

			caller: studioCaller,
			params: codeGrantParams(func(params url.Values) { params.Del("redirect_uri") }),

			daoSelectMock: &respMock[*dao.AuthorizationCode]{resp: changed(validCode, func(code *dao.AuthorizationCode) {
				code.RedirectURIInRequest = false
			})},
			daoRevocationExistMock: &respMock[bool]{},
			daoCredentialsMock:     &respMock[*dao.Credentials]{resp: credentials},
			signRefreshMock:        &errMock{},
			signAccessMock:         &errMock{},
			daoConsumeMock:         &errMock{},

			expectRevocationJTIs: []string{"session-id"},
			expect: &core.TokenGrantResult{
				AccessToken: "access-token", ExpiresIn: accessTokenTTL, RefreshToken: refreshToken,
			},
		},
		{
			// A server that verifies no service token sees no caller, and reads nothing.
			name: "Error/NoCaller",

			params: codeGrantParams(nil),

			expectErr: core.ErrTokenGrantInvalidClient,
		},
		{
			name: "Error/CallerOutsideRegistry",

			caller: accountPlatformCaller,
			params: codeGrantParams(nil),

			expectErr: core.ErrTokenGrantInvalidClient,
		},
		{
			name: "Error/ClientMismatch",

			caller: studioCaller,
			params: codeGrantParams(func(params url.Values) { params.Set("client_id", "forum") }),

			expectErr: core.ErrTokenGrantClientMismatch,
		},
		{
			name: "Error/RepeatedParameter",

			caller: studioCaller,
			params: codeGrantParams(func(params url.Values) { params.Add("code", authorizationCodeSecret) }),

			expectErr: core.ErrOAuthRepeatedParameter,
		},
		{
			name: "Error/NoGrantType",

			caller: studioCaller,
			params: codeGrantParams(func(params url.Values) { params.Del("grant_type") }),

			expectErr: core.ErrTokenGrantNoGrantType,
		},
		{
			name: "Error/UnsupportedGrantType",

			caller: studioCaller,
			params: codeGrantParams(func(params url.Values) { params.Set("grant_type", "password") }),

			expectErr: core.ErrTokenGrantUnsupportedGrantType,
		},
		{
			name: "Error/NoCode",

			caller: studioCaller,
			params: codeGrantParams(func(params url.Values) { params.Del("code") }),

			expectErr: core.ErrTokenGrantNoCode,
		},
		{
			name: "Error/NoCodeVerifier",

			caller: studioCaller,
			params: codeGrantParams(func(params url.Values) { params.Del("code_verifier") }),

			expectErr: core.ErrTokenGrantNoCodeVerifier,
		},
		{
			name: "Error/CodeVerifierTooShort",

			caller: studioCaller,
			params: codeGrantParams(func(params url.Values) { params.Set("code_verifier", rfc7636Verifier[:42]) }),

			expectErr: core.ErrTokenGrantMalformedCodeVerifier,
		},
		{
			// A verifier holds unreserved characters alone (RFC 7636 §4.1).
			name: "Error/CodeVerifierAlphabet",

			caller: studioCaller,
			params: codeGrantParams(func(params url.Values) { params.Set("code_verifier", strings.Repeat("+", 43)) }),

			expectErr: core.ErrTokenGrantMalformedCodeVerifier,
		},
		{
			name: "Error/UnknownCode",

			caller: studioCaller,
			params: codeGrantParams(nil),

			daoSelectMock: &respMock[*dao.AuthorizationCode]{err: dao.ErrAuthorizationCodeSelectNotFound},

			expectErr: core.ErrTokenGrantUnknownCode,
		},
		{
			name: "Error/Select",

			caller: studioCaller,
			params: codeGrantParams(nil),

			daoSelectMock: &respMock[*dao.AuthorizationCode]{err: errFoo},

			expectErr: errFoo,
		},
		{
			// A second exchange ends the session the first one extended to the client.
			name: "Error/Reused",

			caller: studioCaller,
			params: codeGrantParams(nil),

			daoSelectMock:           &respMock[*dao.AuthorizationCode]{resp: consumedCode},
			daoRevocationInsertMock: &errMock{},

			expectErr: core.ErrTokenGrantCodeReused,
		},
		{
			// Reuse is checked first: a replay from another client, or with a wrong verifier, also ends
			// the session.
			name: "Error/ReusedByOtherClient",

			caller: "system:serviceaccount:forum:web",
			params: codeGrantParams(func(params url.Values) { params.Set("code_verifier", strings.Repeat("v", 43)) }),

			daoSelectMock:           &respMock[*dao.AuthorizationCode]{resp: consumedCode},
			daoRevocationInsertMock: &errMock{},

			expectErr: core.ErrTokenGrantCodeReused,
		},
		{
			name: "Error/ReusedRevocation",

			caller: studioCaller,
			params: codeGrantParams(nil),

			daoSelectMock:           &respMock[*dao.AuthorizationCode]{resp: consumedCode},
			daoRevocationInsertMock: &errMock{err: errFoo},

			expectErr:    errFoo,
			expectNotErr: core.ErrTokenGrantCodeReused,
		},
		{
			// Another exchange consumed the code between its read and its consumption.
			name: "Error/ReusedConcurrently",

			caller: studioCaller,
			params: codeGrantParams(nil),

			daoSelectMock:           &respMock[*dao.AuthorizationCode]{resp: &validCode},
			daoRevocationExistMock:  &respMock[bool]{},
			daoCredentialsMock:      &respMock[*dao.Credentials]{resp: credentials},
			signRefreshMock:         &errMock{},
			signAccessMock:          &errMock{},
			daoConsumeMock:          &errMock{err: dao.ErrAuthorizationCodeConsumeNotFound},
			daoRevocationInsertMock: &errMock{},

			expectRevocationJTIs: []string{"session-id"},
			expectErr:            core.ErrTokenGrantCodeReused,
		},
		{
			name: "Error/Expired",

			caller: studioCaller,
			params: codeGrantParams(nil),

			daoSelectMock: &respMock[*dao.AuthorizationCode]{resp: changed(validCode, func(code *dao.AuthorizationCode) {
				code.ExpiresAt = now.Add(-time.Second)
			})},

			expectErr: core.ErrTokenGrantCodeExpired,
		},
		{
			// A code issued to forum, presented by studio.
			name: "Error/ForeignCode",

			caller: studioCaller,
			params: codeGrantParams(nil),

			daoSelectMock: &respMock[*dao.AuthorizationCode]{resp: changed(validCode, func(code *dao.AuthorizationCode) {
				code.ClientID = "forum"
				code.RedirectURI = forumRedirectURI
			})},

			expectErr: core.ErrTokenGrantForeignCode,
		},
		{
			// The authorization request named the redirect URI, so the token request must too.
			name: "Error/NoRedirectURI",

			caller: studioCaller,
			params: codeGrantParams(func(params url.Values) { params.Del("redirect_uri") }),

			daoSelectMock: &respMock[*dao.AuthorizationCode]{resp: &validCode},

			expectErr: core.ErrTokenGrantNoRedirectURI,
		},
		{
			name: "Error/RedirectURIQuery",

			caller: studioCaller,
			params: codeGrantParams(func(params url.Values) { params.Set("redirect_uri", studioRedirectURI+"?next=/") }),

			daoSelectMock: &respMock[*dao.AuthorizationCode]{resp: &validCode},

			expectErr: core.ErrTokenGrantRedirectURIMismatch,
		},
		{
			name: "Error/RedirectURIPrefix",

			caller: studioCaller,
			params: codeGrantParams(func(params url.Values) { params.Set("redirect_uri", "https://studio.example.com/auth") }),

			daoSelectMock: &respMock[*dao.AuthorizationCode]{resp: &validCode},

			expectErr: core.ErrTokenGrantRedirectURIMismatch,
		},
		{
			// A token request that names a redirect URI the authorization request omitted names the
			// one the code was sent to.
			name: "Error/RedirectURIWhenOmitted",

			caller: studioCaller,
			params: codeGrantParams(func(params url.Values) { params.Set("redirect_uri", forumRedirectURI) }),

			daoSelectMock: &respMock[*dao.AuthorizationCode]{resp: changed(validCode, func(code *dao.AuthorizationCode) {
				code.RedirectURIInRequest = false
			})},

			expectErr: core.ErrTokenGrantRedirectURIMismatch,
		},
		{
			name: "Error/WrongVerifier",

			caller: studioCaller,
			params: codeGrantParams(func(params url.Values) { params.Set("code_verifier", strings.Repeat("v", 43)) }),

			daoSelectMock: &respMock[*dao.AuthorizationCode]{resp: &validCode},

			expectErr: core.ErrTokenGrantCodeVerifierMismatch,
		},
		{
			// The plain method sends the challenge as the verifier, which S256 refuses.
			name: "Error/PlainVerifier",

			caller: studioCaller,
			params: codeGrantParams(func(params url.Values) { params.Set("code_verifier", rfc7636Challenge) }),

			daoSelectMock: &respMock[*dao.AuthorizationCode]{resp: &validCode},

			expectErr: core.ErrTokenGrantCodeVerifierMismatch,
		},
		{
			name: "Error/SessionExpired",

			caller: studioCaller,
			params: codeGrantParams(nil),

			daoSelectMock: &respMock[*dao.AuthorizationCode]{resp: changed(validCode, func(code *dao.AuthorizationCode) {
				code.SessionExpiresAt = now.Add(-time.Second)
			})},

			expectErr: core.ErrTokenGrantSessionEnded,
		},
		{
			// The person signed out on another platform after the code was issued.
			name: "Error/SessionRevoked",

			caller: studioCaller,
			params: codeGrantParams(nil),

			daoSelectMock:          &respMock[*dao.AuthorizationCode]{resp: &validCode},
			daoRevocationExistMock: &respMock[bool]{resp: true},

			expectRevocationJTIs: []string{"session-id"},
			expectErr:            core.ErrTokenGrantSessionEnded,
		},
		{
			name: "Error/RevocationExist",

			caller: studioCaller,
			params: codeGrantParams(nil),

			daoSelectMock:          &respMock[*dao.AuthorizationCode]{resp: &validCode},
			daoRevocationExistMock: &respMock[bool]{err: errFoo},

			expectRevocationJTIs: []string{"session-id"},
			expectErr:            errFoo,
		},
		{
			// A grant signs in an existing account, and never creates one.
			name: "Error/UnknownAccount",

			caller: studioCaller,
			params: codeGrantParams(nil),

			daoSelectMock:          &respMock[*dao.AuthorizationCode]{resp: &validCode},
			daoRevocationExistMock: &respMock[bool]{},
			daoCredentialsMock:     &respMock[*dao.Credentials]{err: dao.ErrCredentialsSelectNotFound},

			expectRevocationJTIs: []string{"session-id"},
			expectErr:            core.ErrTokenGrantUnknownCode,
		},
		{
			// A failed signature leaves the code unconsumed, so the client may retry.
			name: "Error/Sign",

			caller: studioCaller,
			params: codeGrantParams(nil),

			daoSelectMock:          &respMock[*dao.AuthorizationCode]{resp: &validCode},
			daoRevocationExistMock: &respMock[bool]{},
			daoCredentialsMock:     &respMock[*dao.Credentials]{resp: credentials},
			signRefreshMock:        &errMock{err: errFoo},

			expectRevocationJTIs: []string{"session-id"},
			expectErr:            errFoo,
		},
		{
			name: "Error/Consume",

			caller: studioCaller,
			params: codeGrantParams(nil),

			daoSelectMock:          &respMock[*dao.AuthorizationCode]{resp: &validCode},
			daoRevocationExistMock: &respMock[bool]{},
			daoCredentialsMock:     &respMock[*dao.Credentials]{resp: credentials},
			signRefreshMock:        &errMock{},
			signAccessMock:         &errMock{},
			daoConsumeMock:         &errMock{err: errFoo},

			expectRevocationJTIs: []string{"session-id"},
			expectErr:            errFoo,
			expectNotErr:         core.ErrTokenGrantCodeReused,
		},
		{
			// A refresh issues a new access token, and keeps the client's refresh token.
			name: "Success/RefreshToken",

			caller: studioCaller,
			params: refreshGrantParams(nil),

			verifyRefreshMock:      &respMock[*session.RefreshClaims]{resp: &refreshTokenClaims},
			daoRevocationExistMock: &respMock[bool]{},
			daoCredentialsMock:     &respMock[*dao.Credentials]{resp: credentials},
			signAccessMock:         &errMock{},

			expectRevocationJTIs: []string{"refresh-token-id", "session-id"},
			expect:               &core.TokenGrantResult{AccessToken: "access-token", ExpiresIn: accessTokenTTL},
		},
		{
			name: "Error/NoRefreshToken",

			caller: studioCaller,
			params: refreshGrantParams(func(params url.Values) { params.Del("refresh_token") }),

			expectErr: core.ErrTokenGrantNoRefreshToken,
		},
		{
			name: "Error/RefreshScope",

			caller: studioCaller,
			params: refreshGrantParams(func(params url.Values) { params.Set("scope", "openid") }),

			expectErr: core.ErrTokenGrantScope,
		},
		{
			name: "Error/InvalidRefreshToken",

			caller: studioCaller,
			params: refreshGrantParams(nil),

			verifyRefreshMock: &respMock[*session.RefreshClaims]{err: jws.ErrInvalidSignature},

			expectErr: core.ErrTokenGrantInvalidRefreshToken,
		},
		{
			// A verifier failure that is not a rejection, such as an unreachable key source, is a fault.
			name: "Error/VerifyRefreshToken",

			caller: studioCaller,
			params: refreshGrantParams(nil),

			verifyRefreshMock: &respMock[*session.RefreshClaims]{err: errFoo},

			expectErr:    errFoo,
			expectNotErr: core.ErrTokenGrantInvalidRefreshToken,
		},
		{
			// The account platform's refresh token was issued to no client (RFC 6749 §10.4).
			name: "Error/AccountPlatformRefreshToken",

			caller: studioCaller,
			params: refreshGrantParams(nil),

			verifyRefreshMock: &respMock[*session.RefreshClaims]{
				resp: changed(refreshTokenClaims, func(claims *session.RefreshClaims) { claims.ClientID = "" }),
			},

			expectErr: core.ErrTokenGrantForeignRefreshToken,
		},
		{
			name: "Error/OtherClientRefreshToken",

			caller: studioCaller,
			params: refreshGrantParams(nil),

			verifyRefreshMock: &respMock[*session.RefreshClaims]{
				resp: changed(refreshTokenClaims, func(claims *session.RefreshClaims) { claims.ClientID = "forum" }),
			},

			expectErr: core.ErrTokenGrantForeignRefreshToken,
		},
		{
			name: "Error/RefreshSessionRevoked",

			caller: studioCaller,
			params: refreshGrantParams(nil),

			verifyRefreshMock:      &respMock[*session.RefreshClaims]{resp: &refreshTokenClaims},
			daoRevocationExistMock: &respMock[bool]{resp: true},

			expectRevocationJTIs: []string{"refresh-token-id", "session-id"},
			expectErr:            core.ErrTokenGrantSessionEnded,
		},
		{
			// The client's refresh token stops with the session it joined, though it has not expired.
			name: "Error/RefreshSessionExpired",

			caller: studioCaller,
			params: refreshGrantParams(nil),

			verifyRefreshMock: &respMock[*session.RefreshClaims]{
				resp: changed(refreshTokenClaims, func(claims *session.RefreshClaims) {
					claims.SessionExp = now.Add(-time.Second).Unix()
				}),
			},
			daoRevocationExistMock: &respMock[bool]{},

			expectRevocationJTIs: []string{"refresh-token-id", "session-id"},
			expectErr:            core.ErrTokenGrantSessionEnded,
		},
		{
			name: "Error/RefreshUnknownAccount",

			caller: studioCaller,
			params: refreshGrantParams(nil),

			verifyRefreshMock:      &respMock[*session.RefreshClaims]{resp: &refreshTokenClaims},
			daoRevocationExistMock: &respMock[bool]{},
			daoCredentialsMock:     &respMock[*dao.Credentials]{err: dao.ErrCredentialsSelectNotFound},

			expectRevocationJTIs: []string{"refresh-token-id", "session-id"},
			expectErr:            core.ErrTokenGrantInvalidRefreshToken,
		},
		{
			name: "Error/RefreshSign",

			caller: studioCaller,
			params: refreshGrantParams(nil),

			verifyRefreshMock:      &respMock[*session.RefreshClaims]{resp: &refreshTokenClaims},
			daoRevocationExistMock: &respMock[bool]{},
			daoCredentialsMock:     &respMock[*dao.Credentials]{resp: credentials},
			signAccessMock:         &errMock{err: errFoo},

			expectRevocationJTIs: []string{"refresh-token-id", "session-id"},
			expectErr:            errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			daoSelect := coremocks.NewMockTokenGrantDaoSelect(t)
			daoConsume := coremocks.NewMockTokenGrantDaoConsume(t)
			daoCredentials := coremocks.NewMockTokenGrantDaoCredentials(t)
			daoRevocationExist := coremocks.NewMockTokenGrantDaoRevocationExist(t)
			daoRevocationInsert := coremocks.NewMockTokenGrantDaoRevocationInsert(t)
			serviceSignClaims := coremocks.NewMockTokenGrantServiceSignClaims(t)
			serviceVerifyRefreshClaims := coremocks.NewMockTokenGrantServiceVerifyRefreshClaims(t)

			if testCase.daoSelectMock != nil {
				daoSelect.EXPECT().
					Exec(mock.Anything, &dao.AuthorizationCodeSelectRequest{ID: codeID[:]}).
					Return(testCase.daoSelectMock.resp, testCase.daoSelectMock.err)
			}

			if testCase.verifyRefreshMock != nil {
				verifyRequest := &servicejsonkeys.VerifyClaimsRequest{
					Usage:       servicejsonkeys.KeyUsageAuthRefresh,
					AccessToken: "refresh-token",
				}

				serviceVerifyRefreshClaims.EXPECT().
					VerifyClaims(mock.Anything, verifyRequest).
					Return(testCase.verifyRefreshMock.resp, testCase.verifyRefreshMock.err)
			}

			if testCase.daoRevocationExistMock != nil {
				daoRevocationExist.EXPECT().
					Exec(mock.Anything, &dao.RefreshTokenRevocationExistRequest{JTIs: testCase.expectRevocationJTIs}).
					Return(testCase.daoRevocationExistMock.resp, testCase.daoRevocationExistMock.err)
			}

			if testCase.daoCredentialsMock != nil {
				daoCredentials.EXPECT().
					Exec(mock.Anything, &dao.CredentialsSelectRequest{ID: userID}).
					Return(testCase.daoCredentialsMock.resp, testCase.daoCredentialsMock.err)
			}

			if testCase.signRefreshMock != nil {
				serviceSignClaims.EXPECT().
					ClaimsSign(mock.Anything, &servicejsonkeys.ClaimsSignRequest{
						Usage:   servicejsonkeys.KeyUsageAuthRefresh,
						Payload: lo.Must(grpcf.MarshalJSONAsAny(refreshTokenForm)),
					}).
					Return(&servicejsonkeys.ClaimsSignResponse{Token: refreshToken}, testCase.signRefreshMock.err)
			}

			if testCase.signAccessMock != nil {
				serviceSignClaims.EXPECT().
					ClaimsSign(mock.Anything, &servicejsonkeys.ClaimsSignRequest{
						Usage:   servicejsonkeys.KeyUsageAuth,
						Payload: lo.Must(grpcf.MarshalJSONAsAny(accessTokenClaims)),
					}).
					Return(&servicejsonkeys.ClaimsSignResponse{Token: "access-token"}, testCase.signAccessMock.err)
			}

			if testCase.daoConsumeMock != nil {
				daoConsume.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(request *dao.AuthorizationCodeConsumeRequest) bool {
						return string(request.ID) == string(codeID[:]) && time.Since(request.Now) < time.Minute
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

			service := core.NewTokenGrant(
				daoSelect, daoConsume, daoCredentials, daoRevocationExist, daoRevocationInsert, serviceSignClaims,
				serviceVerifyRefreshClaims, oauthPreset, accessTokenTTL, leeway,
			)

			res, err := service.Exec(t.Context(), &core.TokenGrantRequest{Caller: testCase.caller, Params: testCase.params})
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)

			if testCase.expectNotErr != nil {
				require.NotErrorIs(t, err, testCase.expectNotErr)
			}

			daoSelect.AssertExpectations(t)
			daoConsume.AssertExpectations(t)
			daoCredentials.AssertExpectations(t)
			daoRevocationExist.AssertExpectations(t)
			daoRevocationInsert.AssertExpectations(t)
			serviceSignClaims.AssertExpectations(t)
			serviceVerifyRefreshClaims.AssertExpectations(t)
		})
	}
}

// authorizationCodeSecret is the code the token grant tests present.
const authorizationCodeSecret = "SplxlOBeZQQYbYS6WxSbIA-cRM_mJ6eoJkHvd3Zm3zQ"

// codeGrantParams returns the parameters of a valid code grant from studio, changed by update when
// it is set.
func codeGrantParams(update func(params url.Values)) url.Values {
	params := url.Values{
		"grant_type":    {core.GrantTypeAuthorizationCode},
		"code":          {authorizationCodeSecret},
		"code_verifier": {rfc7636Verifier},
		"redirect_uri":  {studioRedirectURI},
	}

	if update != nil {
		update(params)
	}

	return params
}

// refreshGrantParams returns the parameters of a valid refresh, changed by update when it is set.
func refreshGrantParams(update func(params url.Values)) url.Values {
	params := url.Values{"grant_type": {core.GrantTypeRefreshToken}, "refresh_token": {"refresh-token"}}

	if update != nil {
		update(params)
	}

	return params
}
