package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/core"
)

// RestAuthorizationRequestValidateService is the core operation GET /v2/authorization-request
// delegates to.
type RestAuthorizationRequestValidateService interface {
	Exec(ctx context.Context, request *core.AuthorizationRequestValidateRequest) error
}

// authorizationRequestErrors maps a refused authorization request to its status. An unknown client
// and another redirect URI both answer 404: no registered client matches.
var authorizationRequestErrors = httpf.ErrMap{
	core.ErrAuthorizationRequestForbiddenCaller:     http.StatusForbidden,
	core.ErrAuthorizationRequestUnknownClient:       http.StatusNotFound,
	core.ErrAuthorizationRequestRedirectURIMismatch: http.StatusNotFound,
	core.ErrInvalidRequest:                          http.StatusUnprocessableEntity,
}

// RestAuthorizationRequestValidateRequest holds the query parameters of GET
// /v2/authorization-request, under the names RFC 6749 §4.1.1 gives them.
type RestAuthorizationRequestValidateRequest struct {
	ClientID    string `schema:"client_id"`
	RedirectURI string `schema:"redirect_uri"`
}

// RestAuthorizationRequestValidate serves GET /v2/authorization-request, which tells the account
// platform whether an authorization request names a client and its redirect URI. It answers an
// empty 204 when it does.
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

	var request RestAuthorizationRequestValidateRequest

	err := muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	caller, _ := httpf.ServiceTokenCaller(ctx)

	serviceRequest := &core.AuthorizationRequestValidateRequest{
		Caller:      caller,
		ClientID:    request.ClientID,
		RedirectURI: request.RedirectURI,
	}

	err = handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, authorizationRequestErrors, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
