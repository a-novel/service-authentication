package api

import (
	"context"
	"fmt"

	"github.com/a-novel/service-authentication/models/api"
	"github.com/a-novel/service-authentication/models/config"
	"github.com/a-novel/service-authentication/pkg"
)

type SecurityHandler struct {
	handler *pkg.HandleBearerAuth[apimodels.OperationName]
}

func NewSecurity(
	source pkg.AuthenticateSource, permissions config.Permissions,
) (*SecurityHandler, error) {
	handler, err := pkg.NewHandleBearerAuth[apimodels.OperationName](source, permissions)
	if err != nil {
		return nil, fmt.Errorf("NewSecurity: %w", err)
	}

	return &SecurityHandler{handler: handler}, nil
}

func (security *SecurityHandler) HandleBearerAuth(
	ctx context.Context, operationName apimodels.OperationName, auth apimodels.BearerAuth,
) (context.Context, error) {
	return security.handler.HandleBearerAuth(ctx, operationName, &auth)
}
