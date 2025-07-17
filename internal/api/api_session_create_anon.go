package api

import (
	"context"
	"fmt"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel/service-authentication/models/api"
)

type LoginAnonService interface {
	LoginAnon(ctx context.Context) (string, error)
}

func (api *API) CreateAnonSession(ctx context.Context) (apimodels.CreateAnonSessionRes, error) {
	ctx, span := otel.Tracer().Start(ctx, "api.CreateAnonSession")
	defer span.End()

	accessToken, err := api.LoginAnonService.LoginAnon(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("login anonymous user: %w", err))
	}

	return otel.ReportSuccess(span, &apimodels.Token{AccessToken: accessToken}), nil
}
