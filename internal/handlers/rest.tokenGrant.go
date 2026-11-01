package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/core"
)

// tokenTypeBearer is the type of every access token the token endpoint issues (RFC 6750).
const tokenTypeBearer = "Bearer"

// RestTokenGrantService is the core operation POST /v2/oauth/token delegates to.
type RestTokenGrantService interface {
	Exec(ctx context.Context, request *core.TokenGrantRequest) (*core.TokenGrantResult, error)
}

// RestTokenResponse is the JSON body of a successful token response (RFC 6749 §5.1).
//
//nolint:tagliatelle // RFC 6749 names the members.
type RestTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	// ExpiresIn is the lifetime of the access token, in seconds.
	ExpiresIn int64 `json:"expires_in"`
	// RefreshToken is absent from a refresh, which issues none.
	RefreshToken string `json:"refresh_token,omitempty"`
}

// RestTokenGrant serves POST /v2/oauth/token, the OAuth 2.0 token endpoint of the clients
// (RFC 6749 §3.2). The client authenticates with its service token, and the form-encoded body is the
// token request's parameters. Every refusal answers its OAuth 2.0 error response.
type RestTokenGrant struct {
	service RestTokenGrantService
	logger  logging.Log
}

func NewRestTokenGrant(service RestTokenGrantService, logger logging.Log) *RestTokenGrant {
	return &RestTokenGrant{service: service, logger: logger}
}

func (handler *RestTokenGrant) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.TokenGrant")
	defer span.End()

	params, err := readOAuthForm(r)
	if err != nil {
		sendOAuthError(w, span, err)

		return
	}

	caller, _ := httpf.ServiceTokenCaller(ctx)

	res, err := handler.service.Exec(ctx, &core.TokenGrantRequest{Caller: caller, Params: params})
	if err != nil {
		if !sendOAuthError(w, span, err) {
			httpf.HandleError(ctx, handler.logger, w, span, nil, err)
		}

		return
	}

	setNoStore(w)
	httpf.SendJSONStatus(w, span, http.StatusOK, RestTokenResponse{
		AccessToken:  res.AccessToken,
		TokenType:    tokenTypeBearer,
		ExpiresIn:    int64(res.ExpiresIn.Seconds()),
		RefreshToken: res.RefreshToken,
	})
}
