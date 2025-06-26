package api

import (
	"context"
	"fmt"

	"github.com/getsentry/sentry-go"

	"github.com/a-novel/service-authentication/api/codegen"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
)

type RequestPasswordResetService interface {
	RequestPasswordReset(ctx context.Context, request services.RequestPasswordResetRequest) (*models.ShortCode, error)
}

func (api *API) RequestPasswordReset(
	ctx context.Context, req *codegen.RequestPasswordResetForm,
) (codegen.RequestPasswordResetRes, error) {
	span := sentry.StartSpan(ctx, "API.RequestPasswordReset")
	defer span.Finish()

	span.SetData("request.email", req.GetEmail())
	span.SetData("request.lang", req.GetLang().Value)

	_, err := api.RequestPasswordResetService.RequestPasswordReset(span.Context(), services.RequestPasswordResetRequest{
		Email: string(req.GetEmail()),
		Lang:  models.Lang(req.GetLang().Value),
	})
	if err != nil {
		span.SetData("service.err", err.Error())

		return nil, fmt.Errorf("request password reset: %w", err)
	}

	return &codegen.RequestPasswordResetNoContent{}, nil
}
