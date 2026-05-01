package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/services"
)

type TokenCreateAnonService interface {
	Exec(ctx context.Context) (*services.Token, error)
}

type TokenCreateAnon struct {
	service TokenCreateAnonService
	logger  logging.Log
}

func NewTokenCreateAnon(service TokenCreateAnonService, logger logging.Log) *TokenCreateAnon {
	return &TokenCreateAnon{service: service, logger: logger}
}

func (handler *TokenCreateAnon) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "handler.TokenCreateAnon")
	defer span.End()

	res, err := handler.service.Exec(ctx)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, nil, err)

		return
	}

	httpf.SendJSON(ctx, w, span, loadToken(res))
}
