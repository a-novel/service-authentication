package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/getsentry/sentry-go"

	"github.com/a-novel/service-authentication/internal/api/codegen"
	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/services"
)

type EmailExistsService interface {
	EmailExists(ctx context.Context, request services.EmailExistsRequest) (bool, error)
}

func (api *API) EmailExists(ctx context.Context, params codegen.EmailExistsParams) (codegen.EmailExistsRes, error) {
	span := sentry.StartSpan(ctx, "API.EmailExists")
	defer span.Finish()

	span.SetData("request.email", params.Email)

	exists, err := api.EmailExistsService.EmailExists(span.Context(), services.EmailExistsRequest{
		Email: string(params.Email),
	})

	span.SetData("service.exists", exists)

	switch {
	case errors.Is(err, dao.ErrCredentialsNotFound):
		span.SetData("service.err", err.Error())

		fallthrough
	case !exists && err == nil:
		return &codegen.NotFoundError{Error: "email not found"}, nil
	case err != nil:
		span.SetData("service.err", err.Error())

		return nil, fmt.Errorf("check email existence: %w", err)
	}

	return &codegen.EmailExistsNoContent{}, nil
}
