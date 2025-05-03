package api

import (
	"fmt"

	"github.com/a-novel-kit/context"

	"github.com/a-novel/service-authentication/api/codegen"
)

type LoginAnonService interface {
	LoginAnon(ctx context.Context) (string, error)
}

func (api *API) CreateAnonSession(ctx context.Context) (codegen.CreateAnonSessionRes, error) {
	accessToken, err := api.LoginAnonService.LoginAnon(ctx)
	if err != nil {
		return nil, fmt.Errorf("login anonymous user: %w", err)
	}

	return &codegen.Token{AccessToken: accessToken}, nil
}
