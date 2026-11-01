package handlers

import (
	"context"
	"net/http"

	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

// RestAuthorizationCodeCreateService is the core operation POST /v2/oauth/authorization-code
// delegates to.
type RestAuthorizationCodeCreateService interface {
	Exec(ctx context.Context, request *core.AuthorizationCodeCreateRequest) (*core.AuthorizationCode, error)
}

// authorizationCodeCreateErrors maps a refusal that is no OAuth 2.0 error response to its status. An
// access token whose session ended answers 401, as an invalid one does.
var authorizationCodeCreateErrors = httpf.ErrMap{
	core.ErrAuthorizationRequestForbiddenCaller: http.StatusForbidden,
	core.ErrAuthorizationCodeCreateSessionEnded: http.StatusUnauthorized,
	core.ErrInvalidRequest:                      http.StatusUnprocessableEntity,
}

// RestAuthorizationCode is the JSON body POST /v2/oauth/authorization-code answers with: the
// authorization response's code, and where to send it (RFC 6749 §4.1.2).
//
//nolint:tagliatelle // RFC 6749 names the members.
type RestAuthorizationCode struct {
	Code        string `json:"code"`
	RedirectURI string `json:"redirect_uri"`
}

// RestAuthorizationCodeCreate serves POST /v2/oauth/authorization-code, which issues a code that
// signs the bearer of the access token in on a client, within their session. The form-encoded body
// is the authorization request's parameters.
type RestAuthorizationCodeCreate struct {
	service RestAuthorizationCodeCreateService
	logger  logging.Log
}

func NewRestAuthorizationCodeCreate(
	service RestAuthorizationCodeCreateService, logger logging.Log,
) *RestAuthorizationCodeCreate {
	return &RestAuthorizationCodeCreate{service: service, logger: logger}
}

func (handler *RestAuthorizationCodeCreate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.AuthorizationCodeCreate")
	defer span.End()

	claims, err := session.MustGetClaimsContext(ctx)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusUnauthorized}, err)

		return
	}

	params, err := readOAuthForm(r)
	if err != nil {
		sendOAuthError(w, span, err)

		return
	}

	caller, _ := httpf.ServiceTokenCaller(ctx)

	serviceRequest := &core.AuthorizationCodeCreateRequest{
		Caller:     caller,
		UserID:     lo.FromPtr(claims.UserID),
		SessionID:  claims.Session(),
		SessionExp: claims.SessionExp,
		Params:     params,
	}

	res, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		if !sendOAuthError(w, span, err) {
			httpf.HandleError(ctx, handler.logger, w, span, authorizationCodeCreateErrors, err)
		}

		return
	}

	setNoStore(w)
	httpf.SendJSONStatus(w, span, http.StatusOK, RestAuthorizationCode{Code: res.Code, RedirectURI: res.RedirectURI})
}
