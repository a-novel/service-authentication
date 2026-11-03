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

// RestSessionListService is the core operation GET /v2/sessions delegates to.
type RestSessionListService interface {
	Exec(ctx context.Context, request *core.SessionListRequest) ([]*core.Session, error)
}

// RestSessionList serves GET /v2/sessions, which lists the caller's sessions, the devices they signed
// in on, active and logged out.
type RestSessionList struct {
	service RestSessionListService
	logger  logging.Log
}

func NewRestSessionList(service RestSessionListService, logger logging.Log) *RestSessionList {
	return &RestSessionList{service: service, logger: logger}
}

func (handler *RestSessionList) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.SessionList")
	defer span.End()

	claims, err := session.MustGetClaimsContext(ctx)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, nil, err)

		return
	}

	serviceRequest := &core.SessionListRequest{
		UserID:    lo.FromPtr(claims.UserID),
		SessionID: claims.Session(),
	}

	res, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, nil, err)

		return
	}

	response := lo.Map(res, func(item *core.Session, _ int) Session { return loadSession(item) })

	httpf.SendJSONStatus(w, span, http.StatusOK, response)
}
