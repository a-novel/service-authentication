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

// RestCredentialsGetSelfService is the core operation GET /v2/credentials/self delegates to.
type RestCredentialsGetSelfService interface {
	Exec(ctx context.Context, request *core.CredentialsGetSelfRequest) (*core.CredentialsSelf, error)
}

// credentialsGetSelfErrors maps a failed lookup of the caller's account to its status.
var credentialsGetSelfErrors = httpf.ErrMap{
	core.ErrCredentialsGetNotFound: http.StatusNotFound,
}

// RestCredentialsGetSelf serves GET /v2/credentials/self, which returns the caller's own account.
type RestCredentialsGetSelf struct {
	service RestCredentialsGetSelfService
	logger  logging.Log
}

func NewRestCredentialsGetSelf(service RestCredentialsGetSelfService, logger logging.Log) *RestCredentialsGetSelf {
	return &RestCredentialsGetSelf{service: service, logger: logger}
}

func (handler *RestCredentialsGetSelf) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.CredentialsGetSelf")
	defer span.End()

	claims, err := session.MustGetClaimsContext(ctx)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, nil, err)

		return
	}

	res, err := handler.service.Exec(ctx, &core.CredentialsGetSelfRequest{ID: lo.FromPtr(claims.UserID)})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, credentialsGetSelfErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadCredentialsSelf(res))
}
