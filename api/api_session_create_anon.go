package api

import (
	"context"
	"fmt"
	"github.com/getsentry/sentry-go"

	"github.com/a-novel/service-authentication/api/codegen"
)

type LoginAnonService interface {
	LoginAnon(ctx context.Context) (string, error)
}

func (api *API) CreateAnonSession(ctx context.Context) (codegen.CreateAnonSessionRes, error) {
	span := sentry.StartSpan(ctx, "API.CreateAnonSession")
	defer span.Finish()

	accessToken, err := api.LoginAnonService.LoginAnon(span.Context())
	if err != nil {
		span.SetData("service.err", err.Error())

		return nil, fmt.Errorf("login anonymous user: %w", err)
	}

	return &codegen.Token{AccessToken: accessToken}, nil
}
