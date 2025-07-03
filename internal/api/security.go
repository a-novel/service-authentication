package api

import (
	"context"
	"fmt"

	"github.com/a-novel/service-authentication/internal/api/codegen"
	"github.com/a-novel/service-authentication/models"
	"github.com/a-novel/service-authentication/pkg"
)

type SecurityHandler struct {
	handler *pkg.HandleBearerAuth[codegen.OperationName]
}

func NewSecurity(
	source pkg.AuthenticateSource, permissions models.PermissionsConfig,
) (*SecurityHandler, error) {
	handler, err := pkg.NewHandleBearerAuth[codegen.OperationName](source, permissions)
	if err != nil {
		return nil, fmt.Errorf("NewSecurity: %w", err)
	}

	return &SecurityHandler{handler: handler}, nil
}

func (security *SecurityHandler) HandleBearerAuth(
	ctx context.Context, operationName codegen.OperationName, auth codegen.BearerAuth,
) (context.Context, error) {
	return security.handler.HandleBearerAuth(ctx, operationName, &auth)
}
