package core

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/otel"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

// The grant types the token endpoint serves, as its grant_type parameter names them.
const (
	// GrantTypeAuthorizationCode trades an authorization code for a token pair (RFC 6749 §4.1.3).
	GrantTypeAuthorizationCode = "authorization_code"
	// GrantTypeRefreshToken trades a client's refresh token for a new access token (RFC 6749 §6).
	GrantTypeRefreshToken = "refresh_token"
)

// The refusals of a token request that concern the request or the client.
var (
	// ErrTokenGrantInvalidClient is returned by [TokenGrant.Exec] when the caller is no client of the
	// registry. A server that verifies no service token sees no caller, and refuses too.
	ErrTokenGrantInvalidClient = &OAuthError{Code: OAuthErrorInvalidClient, Description: "client authentication failed"}
	// ErrTokenGrantClientMismatch is returned by [TokenGrant.Exec] when the client_id parameter names
	// another client than the one the caller authenticates.
	ErrTokenGrantClientMismatch = &OAuthError{
		Code: OAuthErrorInvalidClient, Description: "client_id is not the authenticated client",
	}
	// ErrTokenGrantNoGrantType is returned by [TokenGrant.Exec] for a request without grant_type.
	ErrTokenGrantNoGrantType = &OAuthError{Code: OAuthErrorInvalidRequest, Description: "grant_type is required"}
	// ErrTokenGrantUnsupportedGrantType is returned by [TokenGrant.Exec] for a grant_type other than
	// [GrantTypeAuthorizationCode] and [GrantTypeRefreshToken].
	ErrTokenGrantUnsupportedGrantType = &OAuthError{
		Code: OAuthErrorUnsupportedGrantType, Description: "grant_type must be authorization_code or refresh_token",
	}
	// ErrTokenGrantNoCode is returned by [TokenGrant.Exec] for a code grant without code.
	ErrTokenGrantNoCode = &OAuthError{Code: OAuthErrorInvalidRequest, Description: "code is required"}
	// ErrTokenGrantNoCodeVerifier is returned by [TokenGrant.Exec] for a code grant without
	// code_verifier (RFC 7636 §4.5).
	ErrTokenGrantNoCodeVerifier = &OAuthError{Code: OAuthErrorInvalidRequest, Description: "code_verifier is required"}
	// ErrTokenGrantMalformedCodeVerifier is returned by [TokenGrant.Exec] for a code_verifier outside
	// RFC 7636 §4.1's 43 to 128 unreserved characters.
	ErrTokenGrantMalformedCodeVerifier = &OAuthError{
		Code: OAuthErrorInvalidRequest, Description: "code_verifier malformed",
	}
	// ErrTokenGrantNoRedirectURI is returned by [TokenGrant.Exec] for a code grant without
	// redirect_uri, when the authorization request named one (RFC 6749 §4.1.3).
	ErrTokenGrantNoRedirectURI = &OAuthError{Code: OAuthErrorInvalidRequest, Description: "redirect_uri is required"}
	// ErrTokenGrantNoRefreshToken is returned by [TokenGrant.Exec] for a refresh without
	// refresh_token.
	ErrTokenGrantNoRefreshToken = &OAuthError{Code: OAuthErrorInvalidRequest, Description: "refresh_token is required"}
	// ErrTokenGrantScope is returned by [TokenGrant.Exec] for a refresh that names a scope: the
	// server granted none (RFC 6749 §6).
	ErrTokenGrantScope = &OAuthError{Code: OAuthErrorInvalidScope, Description: "no scope is granted"}
)

// The refusals of a token request that concern its grant, RFC 6749's invalid_grant. A refusal
// leaves the code as it was, except a reuse by the code's own client, which ends the session.
var (
	// ErrTokenGrantUnknownCode is returned by [TokenGrant.Exec] for a code never issued, purged, or
	// whose account no longer exists.
	ErrTokenGrantUnknownCode = &OAuthError{Code: OAuthErrorInvalidGrant, Description: "unknown authorization code"}
	// ErrTokenGrantCodeReused is returned by [TokenGrant.Exec] for a code already exchanged, presented
	// again by its client with its redirect URI and verifier. The exchange also ends the session the
	// code belongs to, so the token pair the first exchange returned stops working too
	// (RFC 6749 §4.1.2, RFC 9700 §4.2.4).
	ErrTokenGrantCodeReused = &OAuthError{Code: OAuthErrorInvalidGrant, Description: "authorization code reused"}
	// ErrTokenGrantCodeExpired is returned by [TokenGrant.Exec] for a code older than
	// [AuthorizationCodeTTL].
	ErrTokenGrantCodeExpired = &OAuthError{Code: OAuthErrorInvalidGrant, Description: "authorization code expired"}
	// ErrTokenGrantForeignCode is returned by [TokenGrant.Exec] for a code issued to another client.
	ErrTokenGrantForeignCode = &OAuthError{
		Code: OAuthErrorInvalidGrant, Description: "authorization code issued to another client",
	}
	// ErrTokenGrantRedirectURIMismatch is returned by [TokenGrant.Exec] when the redirect_uri differs
	// from the one the code was sent to (RFC 6749 §4.1.3, §10.6).
	ErrTokenGrantRedirectURIMismatch = &OAuthError{
		Code: OAuthErrorInvalidGrant, Description: "redirect uri is not the code's",
	}
	// ErrTokenGrantCodeVerifierMismatch is returned by [TokenGrant.Exec] when the transform of the
	// code verifier is not the code's challenge (RFC 7636 §4.6).
	ErrTokenGrantCodeVerifierMismatch = &OAuthError{Code: OAuthErrorInvalidGrant, Description: "invalid code verifier"}
	// ErrTokenGrantInvalidRefreshToken is returned by [TokenGrant.Exec] for a refresh token the
	// verifier rejects, expiry included, or whose account no longer exists.
	ErrTokenGrantInvalidRefreshToken = &OAuthError{Code: OAuthErrorInvalidGrant, Description: "invalid refresh token"}
	// ErrTokenGrantForeignRefreshToken is returned by [TokenGrant.Exec] for a refresh token issued to
	// another client, or held by the account platform (RFC 6749 §10.4).
	ErrTokenGrantForeignRefreshToken = &OAuthError{
		Code: OAuthErrorInvalidGrant, Description: "refresh token issued to another client",
	}
	// ErrTokenGrantSessionEnded is returned by [TokenGrant.Exec] when the session of the code or the
	// refresh token was signed out or has ended.
	ErrTokenGrantSessionEnded = &OAuthError{Code: OAuthErrorInvalidGrant, Description: "session ended"}
)

// reuseRevocationTimeout bounds the revocation that ends the session of a reused code.
const reuseRevocationTimeout = 5 * time.Second

// codeVerifier matches a PKCE code verifier: 43 to 128 unreserved characters (RFC 7636 §4.1).
var codeVerifier = regexp.MustCompile(`^[A-Za-z0-9._~-]{43,128}$`)

// tokenRequestParams are the parameters of a token request (RFC 6749 §4.1.3, §6, RFC 7636 §4.5).
// The server ignores any other.
var tokenRequestParams = []string{
	"grant_type", "code", "redirect_uri", "client_id", "code_verifier", "refresh_token", "scope",
}

// TokenGrantDaoSelect reads the code a client presents.
type TokenGrantDaoSelect interface {
	Exec(ctx context.Context, request *dao.AuthorizationCodeSelectRequest) (*dao.AuthorizationCode, error)
}

// TokenGrantDaoConsume marks the code exchanged, once.
type TokenGrantDaoConsume interface {
	Exec(ctx context.Context, request *dao.AuthorizationCodeConsumeRequest) error
}

// TokenGrantDaoCredentials loads the account a grant signs in.
type TokenGrantDaoCredentials interface {
	Exec(ctx context.Context, request *dao.CredentialsSelectRequest) (*dao.Credentials, error)
}

// TokenGrantDaoRevocationExist reports whether the grant's session was signed out.
type TokenGrantDaoRevocationExist interface {
	Exec(ctx context.Context, request *dao.RefreshTokenRevocationExistRequest) (bool, error)
}

// TokenGrantDaoRevocationInsert ends the session of a reused code.
type TokenGrantDaoRevocationInsert interface {
	Exec(ctx context.Context, request *dao.RefreshTokenRevocationInsertRequest) error
}

// TokenGrantServiceSignClaims signs the tokens.
type TokenGrantServiceSignClaims interface {
	ClaimsSign(
		ctx context.Context, req *servicejsonkeys.ClaimsSignRequest, opts ...grpc.CallOption,
	) (*servicejsonkeys.ClaimsSignResponse, error)
}

// TokenGrantServiceVerifyRefreshClaims verifies a client's refresh token and decodes its claims.
// Same published-client method name as [TokenRefreshServiceVerifyClaims].
//
// nosemgrep: a-novel-dep-interface-method-must-be-exec
type TokenGrantServiceVerifyRefreshClaims interface {
	VerifyClaims(ctx context.Context, req *servicejsonkeys.VerifyClaimsRequest) (*session.RefreshClaims, error)
}

// TokenGrantRequest carries a request to the token endpoint (RFC 6749 §3.2).
type TokenGrantRequest struct {
	// Caller is the verified Kubernetes caller of the request, which authenticates the client
	// (RFC 6749 §2.3.2). It is empty when the server verifies none.
	Caller string
	// Params are the parameters of the request, from its application/x-www-form-urlencoded body.
	Params url.Values
}

// TokenGrantResult is a successful token response (RFC 6749 §5.1). Its token type is Bearer
// (RFC 6750).
type TokenGrantResult struct {
	AccessToken string
	// ExpiresIn is the lifetime of the access token.
	ExpiresIn time.Duration
	// RefreshToken is the refresh token a code grant issues. A refresh issues none: the client keeps
	// its refresh token.
	RefreshToken string
}

// TokenGrant serves the OAuth 2.0 token endpoint for the clients of the registry, which
// authenticate with their Kubernetes caller. It serves two grants:
//   - authorization_code trades a code for a token pair in the session of the account platform that
//     issued it, once, with the redirect URI and the PKCE verifier the code is bound to. It signs an
//     existing account in, and never creates one.
//   - refresh_token trades the client's refresh token for a new access token, while its session
//     lasts. A refresh token renews only for the client it was issued to (RFC 6749 §10.4).
//
// A refusal is an [OAuthError]. The code grant signs the pair before it consumes the code, so a
// failed signature leaves the code to a retry. The consumption is one conditional write: of two
// concurrent exchanges, the second finds the code consumed and ends the session as any reuse does.
type TokenGrant struct {
	daoSelect                  TokenGrantDaoSelect
	daoConsume                 TokenGrantDaoConsume
	daoCredentials             TokenGrantDaoCredentials
	daoRevocationExist         TokenGrantDaoRevocationExist
	daoRevocationInsert        TokenGrantDaoRevocationInsert
	serviceSignClaims          TokenGrantServiceSignClaims
	serviceVerifyRefreshClaims TokenGrantServiceVerifyRefreshClaims
	oauth                      authconfig.OAuth
	// accessTokenTTL is how long the signer's access tokens last, the response's expires_in.
	accessTokenTTL time.Duration
	// refreshTokenLeeway is the clock-skew tolerance the verifier grants an expired refresh token.
	refreshTokenLeeway time.Duration
}

func NewTokenGrant(
	daoSelect TokenGrantDaoSelect,
	daoConsume TokenGrantDaoConsume,
	daoCredentials TokenGrantDaoCredentials,
	daoRevocationExist TokenGrantDaoRevocationExist,
	daoRevocationInsert TokenGrantDaoRevocationInsert,
	serviceSignClaims TokenGrantServiceSignClaims,
	serviceVerifyRefreshClaims TokenGrantServiceVerifyRefreshClaims,
	oauth authconfig.OAuth,
	accessTokenTTL time.Duration,
	refreshTokenLeeway time.Duration,
) *TokenGrant {
	return &TokenGrant{
		daoSelect:                  daoSelect,
		daoConsume:                 daoConsume,
		daoCredentials:             daoCredentials,
		daoRevocationExist:         daoRevocationExist,
		daoRevocationInsert:        daoRevocationInsert,
		serviceSignClaims:          serviceSignClaims,
		serviceVerifyRefreshClaims: serviceVerifyRefreshClaims,
		oauth:                      oauth,
		accessTokenTTL:             accessTokenTTL,
		refreshTokenLeeway:         refreshTokenLeeway,
	}
}

func (service *TokenGrant) Exec(ctx context.Context, request *TokenGrantRequest) (*TokenGrantResult, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.TokenGrant")
	defer span.End()

	grantType := request.Params.Get("grant_type")

	span.SetAttributes(attribute.String("oauth.grant_type", grantType))

	clientID, ok := service.clientOf(request.Caller)
	if !ok {
		return nil, reportTokenGrant(span, ErrTokenGrantInvalidClient)
	}

	span.SetAttributes(attribute.String("client.id", clientID))

	if oauthParamsRepeated(request.Params, tokenRequestParams...) {
		return nil, reportTokenGrant(span, ErrOAuthRepeatedParameter)
	}

	// A client may name itself, as RFC 6749 §3.2.1 allows: it names the client it authenticates as.
	requestClientID := request.Params.Get("client_id")
	if requestClientID != "" && requestClientID != clientID {
		return nil, reportTokenGrant(span, ErrTokenGrantClientMismatch)
	}

	var (
		result *TokenGrantResult
		err    error
	)

	switch grantType {
	case "":
		err = ErrTokenGrantNoGrantType
	case GrantTypeAuthorizationCode:
		result, err = service.grantAuthorizationCode(ctx, clientID, request.Params)
	case GrantTypeRefreshToken:
		result, err = service.grantRefreshToken(ctx, clientID, request.Params)
	default:
		err = ErrTokenGrantUnsupportedGrantType
	}

	if err != nil {
		return nil, reportTokenGrant(span, err)
	}

	return result, nil
}

// clientOf returns the ID of the client whose caller is caller. No caller names no client.
func (service *TokenGrant) clientOf(caller string) (string, bool) {
	if caller == "" {
		return "", false
	}

	for clientID, client := range service.oauth.Clients {
		if client.Caller == caller {
			return clientID, true
		}
	}

	return "", false
}

// grantAuthorizationCode trades the code params carry for a token pair (RFC 6749 §4.1.3).
func (service *TokenGrant) grantAuthorizationCode(
	ctx context.Context, clientID string, params url.Values,
) (*TokenGrantResult, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.TokenGrant(authorizationCode)")
	defer span.End()

	verifier := params.Get("code_verifier")

	switch {
	case params.Get("code") == "":
		return nil, otel.ReportError(span, ErrTokenGrantNoCode)
	case verifier == "":
		return nil, otel.ReportError(span, ErrTokenGrantNoCodeVerifier)
	case !codeVerifier.MatchString(verifier):
		return nil, otel.ReportError(span, ErrTokenGrantMalformedCodeVerifier)
	}

	now := time.Now()
	id := sha256.Sum256([]byte(params.Get("code")))

	code, err := service.daoSelect.Exec(ctx, &dao.AuthorizationCodeSelectRequest{ID: id[:]})
	if errors.Is(err, dao.ErrAuthorizationCodeSelectNotFound) {
		return nil, otel.ReportError(span, errors.Join(err, ErrTokenGrantUnknownCode))
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select authorization code: %w", err))
	}

	span.SetAttributes(
		attribute.String("user.id", code.UserID.String()),
		attribute.String("session.id", code.SessionID),
	)

	// The binding comes first, so only the code's own client, with its redirect URI and verifier, ends
	// the session by replaying it. Anyone else who holds a used code is refused, and ends nothing.
	err = checkCodeBinding(code, clientID, params)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	if code.ConsumedAt != nil {
		return nil, otel.ReportError(span, service.refuseReuse(ctx, code, now))
	}

	err = checkCodeLifetime(code, now)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	revocationRequest := &dao.RefreshTokenRevocationExistRequest{JTIs: []string{code.SessionID}}

	revoked, err := service.daoRevocationExist.Exec(ctx, revocationRequest)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("check session revocation: %w", err))
	}

	if revoked {
		return nil, otel.ReportError(span, ErrTokenGrantSessionEnded)
	}

	credentials, err := service.daoCredentials.Exec(ctx, &dao.CredentialsSelectRequest{ID: code.UserID})
	if errors.Is(err, dao.ErrCredentialsSelectNotFound) {
		return nil, otel.ReportError(span, errors.Join(err, ErrTokenGrantUnknownCode))
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select credentials: %w", err))
	}

	refreshClaims := session.RefreshClaimsForm{
		SessionID:  code.SessionID,
		SessionExp: code.SessionExpiresAt.Unix(),
		ClientID:   clientID,
	}

	tokens, err := signTokenPair(ctx, service.serviceSignClaims, credentials, refreshClaims)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("sign token pair: %w", err))
	}

	err = service.daoConsume.Exec(ctx, &dao.AuthorizationCodeConsumeRequest{ID: id[:], Now: now})
	if errors.Is(err, dao.ErrAuthorizationCodeConsumeNotFound) {
		// The pair signed here is dropped: another exchange consumed the code since it was read, or the
		// code was deleted.
		return nil, otel.ReportError(span, service.refuseLostCode(ctx, code, now))
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("consume authorization code: %w", err))
	}

	return &TokenGrantResult{
		AccessToken:  tokens.AccessToken,
		ExpiresIn:    service.accessTokenTTL,
		RefreshToken: tokens.RefreshToken,
	}, nil
}

// grantRefreshToken trades the client's refresh token params carry for a new access token
// (RFC 6749 §6).
func (service *TokenGrant) grantRefreshToken(
	ctx context.Context, clientID string, params url.Values,
) (*TokenGrantResult, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.TokenGrant(refreshToken)")
	defer span.End()

	switch {
	case params.Get("refresh_token") == "":
		return nil, otel.ReportError(span, ErrTokenGrantNoRefreshToken)
	case params.Get("scope") != "":
		return nil, otel.ReportError(span, ErrTokenGrantScope)
	}

	verifyRequest := &servicejsonkeys.VerifyClaimsRequest{
		Usage:       servicejsonkeys.KeyUsageAuthRefresh,
		AccessToken: params.Get("refresh_token"),
	}

	claims, err := service.serviceVerifyRefreshClaims.VerifyClaims(ctx, verifyRequest)
	if err != nil {
		return nil, otel.ReportError(span, joinInvalidToken(err, ErrTokenGrantInvalidRefreshToken))
	}

	span.SetAttributes(
		attribute.String("user.id", claims.UserID.String()),
		attribute.String("session.id", claims.Session()),
	)

	if claims.ClientID != clientID {
		return nil, otel.ReportError(span, ErrTokenGrantForeignRefreshToken)
	}

	revocationRequest := &dao.RefreshTokenRevocationExistRequest{JTIs: []string{claims.Jti, claims.Session()}}

	revoked, err := service.daoRevocationExist.Exec(ctx, revocationRequest)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("check session revocation: %w", err))
	}

	if revoked || time.Now().Unix() >= claims.SessionExpiry() {
		return nil, otel.ReportError(span, ErrTokenGrantSessionEnded)
	}

	credentials, err := service.daoCredentials.Exec(ctx, &dao.CredentialsSelectRequest{ID: claims.UserID})
	if errors.Is(err, dao.ErrCredentialsSelectNotFound) {
		return nil, otel.ReportError(span, errors.Join(err, ErrTokenGrantInvalidRefreshToken))
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select credentials: %w", err))
	}

	accessToken, err := signAccessToken(ctx, service.serviceSignClaims, credentials, claims)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	return &TokenGrantResult{AccessToken: accessToken, ExpiresIn: service.accessTokenTTL}, nil
}

// refuseLostCode returns the refusal of a code the exchange read but could not consume. A code
// another exchange consumed is a reuse. A code deleted meanwhile, by the purge or with its account,
// is unknown.
func (service *TokenGrant) refuseLostCode(ctx context.Context, code *dao.AuthorizationCode, now time.Time) error {
	ctx, span := otel.Tracer().Start(ctx, "core.TokenGrant(refuseLostCode)")
	defer span.End()

	_, err := service.daoSelect.Exec(ctx, &dao.AuthorizationCodeSelectRequest{ID: code.ID})
	if errors.Is(err, dao.ErrAuthorizationCodeSelectNotFound) {
		return otel.ReportError(span, errors.Join(err, ErrTokenGrantUnknownCode))
	}

	if err != nil {
		return otel.ReportError(span, fmt.Errorf("select authorization code: %w", err))
	}

	return otel.ReportError(span, service.refuseReuse(ctx, code, now))
}

// refuseReuse ends the session of a reused code, which its first exchange extended to the client,
// and returns the refusal. The revocation lasts as long as a refresh token of the session may. It
// runs on a context of its own, so a caller that leaves does not keep the session alive.
func (service *TokenGrant) refuseReuse(ctx context.Context, code *dao.AuthorizationCode, now time.Time) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), reuseRevocationTimeout)
	defer cancel()

	ctx, span := otel.Tracer().Start(ctx, "core.TokenGrant(refuseReuse)")
	defer span.End()

	revocation := &dao.RefreshTokenRevocationInsertRequest{
		JTIs:      []string{code.SessionID},
		ExpiresAt: code.SessionExpiresAt.Add(service.refreshTokenLeeway),
		Now:       now,
	}

	err := service.daoRevocationInsert.Exec(ctx, revocation)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("revoke session: %w", err))
	}

	return ErrTokenGrantCodeReused
}

// checkCodeBinding returns the refusal of a code issued to another client, or that the redirect URI
// or the code verifier params carry does not match.
func checkCodeBinding(code *dao.AuthorizationCode, clientID string, params url.Values) error {
	redirectURI := params.Get("redirect_uri")

	switch {
	case code.ClientID != clientID:
		return ErrTokenGrantForeignCode
	case code.RedirectURIInRequest && redirectURI == "":
		return ErrTokenGrantNoRedirectURI
	case redirectURI != "" && redirectURI != code.RedirectURI:
		return ErrTokenGrantRedirectURIMismatch
	case !verifiesChallenge(code, params.Get("code_verifier")):
		return ErrTokenGrantCodeVerifierMismatch
	}

	return nil
}

// checkCodeLifetime returns the refusal of a code that expired, or whose session ended.
func checkCodeLifetime(code *dao.AuthorizationCode, now time.Time) error {
	switch {
	case !now.Before(code.ExpiresAt):
		return ErrTokenGrantCodeExpired
	case !now.Before(code.SessionExpiresAt):
		return ErrTokenGrantSessionEnded
	}

	return nil
}

// verifiesChallenge reports whether verifier transforms into the code's challenge, by the code's
// method (RFC 7636 §4.6). It compares in constant time.
func verifiesChallenge(code *dao.AuthorizationCode, verifier string) bool {
	if code.CodeChallengeMethod != CodeChallengeMethodS256 {
		return false
	}

	digest := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(digest[:])

	return subtle.ConstantTimeCompare([]byte(challenge), []byte(code.CodeChallenge)) == 1
}

// reportTokenGrant reports err on span. A refusal also sets the span's oauth.error and
// oauth.error_description attributes, which the Services tab counts per client and grant type.
func reportTokenGrant(span trace.Span, err error) error {
	var refusal *OAuthError
	if errors.As(err, &refusal) {
		span.SetAttributes(
			attribute.String("oauth.error", refusal.Code),
			attribute.String("oauth.error_description", refusal.Description),
		)
	}

	return otel.ReportError(span, err)
}
