package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/core"
)

// RestTokenCreateAnonService is the core operation PUT /v2/session/anon delegates to.
type RestTokenCreateAnonService interface {
	Exec(ctx context.Context) (*core.Token, error)
}

// RestTokenCreateAnon serves PUT /v2/session/anon, which opens an anonymous session.
type RestTokenCreateAnon struct {
	service RestTokenCreateAnonService
	logger  logging.Log
}

func NewRestTokenCreateAnon(service RestTokenCreateAnonService, logger logging.Log) *RestTokenCreateAnon {
	return &RestTokenCreateAnon{service: service, logger: logger}
}

func (handler *RestTokenCreateAnon) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.TokenCreateAnon")
	defer span.End()

	res, err := handler.service.Exec(ctx)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, nil, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadToken(res))
}
