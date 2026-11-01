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

// RestAuthorizationCodeCreateService is the core operation PUT /v2/authorization-code delegates to.
type RestAuthorizationCodeCreateService interface {
	Exec(ctx context.Context, request *core.AuthorizationCodeCreateRequest) (*core.AuthorizationCode, error)
}

// authorizationCodeCreateErrors maps a refused code request to its status, as for an authorization
// request. An access token whose session ended answers 401, as an invalid one does.
var authorizationCodeCreateErrors = httpf.ErrMap{
	core.ErrAuthorizationRequestForbiddenCaller:     http.StatusForbidden,
	core.ErrAuthorizationRequestUnknownClient:       http.StatusNotFound,
	core.ErrAuthorizationRequestRedirectURIMismatch: http.StatusNotFound,
	core.ErrAuthorizationCodeCreateSessionEnded:     http.StatusUnauthorized,
	core.ErrInvalidRequest:                          http.StatusUnprocessableEntity,
}

// RestAuthorizationCodeCreateRequest is the JSON body of PUT /v2/authorization-code. Its members
// keep the names RFC 6749 §4.1.1 and RFC 7636 §4.3 give the parameters of the authorization request.
//
//nolint:tagliatelle // RFC 6749 and RFC 7636 name the members.
type RestAuthorizationCodeCreateRequest struct {
	ClientID            string `json:"client_id"`
	RedirectURI         string `json:"redirect_uri"`
	CodeChallenge       string `json:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method"`
}

// RestAuthorizationCode is the JSON body PUT /v2/authorization-code answers with.
type RestAuthorizationCode struct {
	// Code is the code parameter the account platform adds to the client's redirect URI.
	Code string `json:"code"`
}

// RestAuthorizationCodeCreate serves PUT /v2/authorization-code, which issues a code that signs the
// bearer of the access token in on a client, within their session.
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

	var request RestAuthorizationCodeCreateRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	caller, _ := httpf.ServiceTokenCaller(ctx)

	serviceRequest := &core.AuthorizationCodeCreateRequest{
		Caller:              caller,
		UserID:              lo.FromPtr(claims.UserID),
		SessionID:           claims.Session(),
		SessionExp:          claims.SessionExp,
		ClientID:            request.ClientID,
		RedirectURI:         request.RedirectURI,
		CodeChallenge:       request.CodeChallenge,
		CodeChallengeMethod: request.CodeChallengeMethod,
	}

	res, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, authorizationCodeCreateErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, RestAuthorizationCode{Code: res.Code})
}
