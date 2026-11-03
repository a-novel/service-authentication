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

// RestCredentialsUpdateLocationConsentService is the core operation PATCH
// /v2/credentials/location-consent delegates to.
type RestCredentialsUpdateLocationConsentService interface {
	Exec(ctx context.Context, request *core.CredentialsUpdateLocationConsentRequest) error
}

// credentialsUpdateLocationConsentErrors maps a refused consent update to its status.
var credentialsUpdateLocationConsentErrors = httpf.ErrMap{
	core.ErrCredentialsUpdateLocationConsentNotFound: http.StatusNotFound,
	core.ErrInvalidRequest:                           http.StatusUnprocessableEntity,
}

// RestCredentialsUpdateLocationConsentRequest is the JSON body of PATCH /v2/credentials/location-consent.
type RestCredentialsUpdateLocationConsentRequest struct {
	// Enabled is nil when the body omits it, which the core refuses.
	Enabled *bool `json:"enabled"`
}

// RestCredentialsUpdateLocationConsent serves PATCH /v2/credentials/location-consent, which sets
// whether the caller's sessions record their location. It answers an empty 204.
type RestCredentialsUpdateLocationConsent struct {
	service RestCredentialsUpdateLocationConsentService
	logger  logging.Log
}

func NewRestCredentialsUpdateLocationConsent(
	service RestCredentialsUpdateLocationConsentService, logger logging.Log,
) *RestCredentialsUpdateLocationConsent {
	return &RestCredentialsUpdateLocationConsent{service: service, logger: logger}
}

func (handler *RestCredentialsUpdateLocationConsent) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.CredentialsUpdateLocationConsent")
	defer span.End()

	claims, err := session.MustGetClaimsContext(ctx)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, nil, err)

		return
	}

	var request RestCredentialsUpdateLocationConsentRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.CredentialsUpdateLocationConsentRequest{
		UserID:  lo.FromPtr(claims.UserID),
		Enabled: request.Enabled,
	}

	err = handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, credentialsUpdateLocationConsentErrors, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
