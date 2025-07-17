package api

import (
	"context"
	"fmt"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
	"github.com/a-novel/service-authentication/models/api"
)

type RequestPasswordResetService interface {
	RequestPasswordReset(ctx context.Context, request services.RequestPasswordResetRequest) (*models.ShortCode, error)
}

func (api *API) RequestPasswordReset(
	ctx context.Context, req *apimodels.RequestPasswordResetForm,
) (apimodels.RequestPasswordResetRes, error) {
	ctx, span := otel.Tracer().Start(ctx, "api.RequestPasswordReset")
	defer span.End()

	_, err := api.RequestPasswordResetService.RequestPasswordReset(ctx, services.RequestPasswordResetRequest{
		Email: string(req.GetEmail()),
		Lang:  models.Lang(req.GetLang().Value),
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("request password reset: %w", err))
	}

	return otel.ReportSuccess(span, &apimodels.RequestPasswordResetNoContent{}), nil
}
