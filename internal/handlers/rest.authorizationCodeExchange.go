package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/core"
)

// RestAuthorizationCodeExchangeService is the core operation PUT /v2/session/authorization-code
// delegates to.
type RestAuthorizationCodeExchangeService interface {
	Exec(ctx context.Context, request *core.AuthorizationCodeExchangeRequest) (*core.Token, error)
}

// authorizationCodeExchangeErrors maps a refused exchange to its status. Every refused code answers
// 403, whatever the reason, as RFC 6749's invalid_grant does; the trace records the reason.
var authorizationCodeExchangeErrors = httpf.ErrMap{
	core.ErrAuthorizationCodeExchangeForbiddenCaller:     http.StatusForbidden,
	core.ErrAuthorizationCodeExchangeUnknownCode:         http.StatusForbidden,
	core.ErrAuthorizationCodeExchangeReused:              http.StatusForbidden,
	core.ErrAuthorizationCodeExchangeExpired:             http.StatusForbidden,
	core.ErrAuthorizationCodeExchangeForeignCode:         http.StatusForbidden,
	core.ErrAuthorizationCodeExchangeRedirectURIMismatch: http.StatusForbidden,
	core.ErrAuthorizationCodeExchangeInvalidVerifier:     http.StatusForbidden,
	core.ErrAuthorizationCodeExchangeSessionEnded:        http.StatusForbidden,
	core.ErrInvalidRequest:                               http.StatusUnprocessableEntity,
}

// RestAuthorizationCodeExchangeRequest is the JSON body of PUT /v2/session/authorization-code. Its
// members keep the names RFC 6749 §4.1.3 and RFC 7636 §4.5 give the parameters of the token request.
//
//nolint:tagliatelle // RFC 6749 and RFC 7636 name the members.
type RestAuthorizationCodeExchangeRequest struct {
	Code         string `json:"code"`
	CodeVerifier string `json:"code_verifier"`
	RedirectURI  string `json:"redirect_uri"`
}

// RestAuthorizationCodeExchange serves PUT /v2/session/authorization-code, which signs a person in
// on a client: the client's caller trades a code for a token pair in the session the code belongs to.
type RestAuthorizationCodeExchange struct {
	service RestAuthorizationCodeExchangeService
	logger  logging.Log
}

func NewRestAuthorizationCodeExchange(
	service RestAuthorizationCodeExchangeService, logger logging.Log,
) *RestAuthorizationCodeExchange {
	return &RestAuthorizationCodeExchange{service: service, logger: logger}
}

func (handler *RestAuthorizationCodeExchange) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.AuthorizationCodeExchange")
	defer span.End()

	var request RestAuthorizationCodeExchangeRequest

	err := httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	caller, _ := httpf.ServiceTokenCaller(ctx)

	serviceRequest := &core.AuthorizationCodeExchangeRequest{
		Caller:       caller,
		Code:         request.Code,
		CodeVerifier: request.CodeVerifier,
		RedirectURI:  request.RedirectURI,
	}

	res, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, authorizationCodeExchangeErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadToken(res))
}
