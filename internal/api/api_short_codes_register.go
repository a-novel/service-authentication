package api

import (
	"context"
	"fmt"

	"github.com/getsentry/sentry-go"

	"github.com/a-novel/service-authentication/internal/api/codegen"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
)

type RequestRegisterService interface {
	RequestRegister(ctx context.Context, request services.RequestRegisterRequest) (*models.ShortCode, error)
}

func (api *API) RequestRegistration(
	ctx context.Context, req *codegen.RequestRegistrationForm,
) (codegen.RequestRegistrationRes, error) {
	span := sentry.StartSpan(ctx, "API.RequestRegistration")
	defer span.Finish()

	span.SetData("request.email", req.GetEmail())
	span.SetData("request.lang", req.GetLang().Value)

	_, err := api.RequestRegisterService.RequestRegister(span.Context(), services.RequestRegisterRequest{
		Email: string(req.GetEmail()),
		Lang:  models.Lang(req.GetLang().Value),
	})
	if err != nil {
		span.SetData("service.err", err.Error())

		return nil, fmt.Errorf("request register: %w", err)
	}

	return &codegen.RequestRegistrationNoContent{}, nil
}
