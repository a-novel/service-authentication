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

// RestSessionRevokeOthersService is the core operation PUT /v2/sessions/revoke-others delegates to.
type RestSessionRevokeOthersService interface {
	Exec(ctx context.Context, request *core.SessionRevokeOthersRequest) error
}

// RestSessionRevokeOthers serves PUT /v2/sessions/revoke-others, which signs out every live session of
// the caller but their own. It answers an empty 204.
type RestSessionRevokeOthers struct {
	service RestSessionRevokeOthersService
	logger  logging.Log
}

func NewRestSessionRevokeOthers(service RestSessionRevokeOthersService, logger logging.Log) *RestSessionRevokeOthers {
	return &RestSessionRevokeOthers{service: service, logger: logger}
}

func (handler *RestSessionRevokeOthers) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.SessionRevokeOthers")
	defer span.End()

	claims, err := session.MustGetClaimsContext(ctx)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, nil, err)

		return
	}

	serviceRequest := &core.SessionRevokeOthersRequest{
		UserID:    lo.FromPtr(claims.UserID),
		SessionID: claims.Session(),
	}

	err = handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, nil, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
