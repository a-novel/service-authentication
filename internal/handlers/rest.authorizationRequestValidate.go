package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/core"
)

// RestAuthorizationRequestValidateService is the core operation GET /v2/oauth/authorization-request
// delegates to.
type RestAuthorizationRequestValidateService interface {
	Exec(ctx context.Context, request *core.AuthorizationRequestValidateRequest) (*core.AuthorizationRequest, error)
}

// authorizationRequestErrors maps a refused caller to its status. A refused request answers its
// OAuth 2.0 error response instead.
var authorizationRequestErrors = httpf.ErrMap{
	core.ErrAuthorizationRequestForbiddenCaller: http.StatusForbidden,
}

// RestAuthorizationRequest is the JSON body GET /v2/oauth/authorization-request answers with.
//
//nolint:tagliatelle // RFC 6749 names the member.
type RestAuthorizationRequest struct {
	// RedirectURI is where the account platform sends the response once the person signed in.
	RedirectURI string `json:"redirect_uri"`
}

// RestAuthorizationRequestValidate serves GET /v2/oauth/authorization-request, which checks an
// authorization request for the account platform. The query string is the authorization request's.
type RestAuthorizationRequestValidate struct {
	service RestAuthorizationRequestValidateService
	logger  logging.Log
}

func NewRestAuthorizationRequestValidate(
	service RestAuthorizationRequestValidateService, logger logging.Log,
) *RestAuthorizationRequestValidate {
	return &RestAuthorizationRequestValidate{service: service, logger: logger}
}

func (handler *RestAuthorizationRequestValidate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.AuthorizationRequestValidate")
	defer span.End()

	params, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		sendOAuthError(w, span, errors.Join(err, core.ErrOAuthMalformedRequest))

		return
	}

	caller, _ := httpf.ServiceTokenCaller(ctx)

	res, err := handler.service.Exec(ctx, &core.AuthorizationRequestValidateRequest{Caller: caller, Params: params})
	if err != nil {
		if !sendOAuthError(w, span, err) {
			httpf.HandleError(ctx, handler.logger, w, span, authorizationRequestErrors, err)
		}

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, RestAuthorizationRequest{RedirectURI: res.RedirectURI})
}
