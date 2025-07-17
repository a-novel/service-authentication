package api

import (
	"context"
	"fmt"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
	"github.com/a-novel/service-authentication/models/api"
)

type RequestRegisterService interface {
	RequestRegister(ctx context.Context, request services.RequestRegisterRequest) (*models.ShortCode, error)
}

func (api *API) RequestRegistration(
	ctx context.Context, req *apimodels.RequestRegistrationForm,
) (apimodels.RequestRegistrationRes, error) {
	ctx, span := otel.Tracer().Start(ctx, "api.RequestRegistration")
	defer span.End()

	_, err := api.RequestRegisterService.RequestRegister(ctx, services.RequestRegisterRequest{
		Email: string(req.GetEmail()),
		Lang:  models.Lang(req.GetLang().Value),
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("request register: %w", err))
	}

	return otel.ReportSuccess(span, &apimodels.RequestRegistrationNoContent{}), nil
}
