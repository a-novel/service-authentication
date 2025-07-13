package api

import (
	"context"
	"fmt"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
	"github.com/a-novel/service-authentication/models/api"
	"github.com/a-novel/service-authentication/pkg"
)

type RequestEmailUpdateService interface {
	RequestEmailUpdate(ctx context.Context, request services.RequestEmailUpdateRequest) (*models.ShortCode, error)
}

func (api *API) RequestEmailUpdate(
	ctx context.Context, req *apimodels.RequestEmailUpdateForm,
) (apimodels.RequestEmailUpdateRes, error) {
	ctx, span := otel.Tracer().Start(ctx, "api.RequestEmailUpdate")
	defer span.End()

	userID, err := pkg.RequireUserID(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("require user ID: %w", err))
	}

	_, err = api.RequestEmailUpdateService.RequestEmailUpdate(ctx, services.RequestEmailUpdateRequest{
		Email: string(req.GetEmail()),
		Lang:  models.Lang(req.GetLang().Value),
		ID:    userID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("request email update: %w", err))
	}

	return otel.ReportSuccess(span, &apimodels.RequestEmailUpdateNoContent{}), nil
}
