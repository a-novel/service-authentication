package api

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/codes"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models/api"
)

type EmailExistsService interface {
	EmailExists(ctx context.Context, request services.EmailExistsRequest) (bool, error)
}

func (api *API) EmailExists(ctx context.Context, params apimodels.EmailExistsParams) (apimodels.EmailExistsRes, error) {
	ctx, span := otel.Tracer().Start(ctx, "api.EmailExists")
	defer span.End()

	exists, err := api.EmailExistsService.EmailExists(ctx, services.EmailExistsRequest{
		Email: string(params.Email),
	})

	switch {
	case errors.Is(err, dao.ErrCredentialsNotFound) || (!exists && err == nil):
		span.RecordError(err)
		span.SetStatus(codes.Error, "")

		return &apimodels.NotFoundError{Error: "email not found"}, nil
	case err != nil:
		span.RecordError(err)
		span.SetStatus(codes.Error, "")

		return nil, fmt.Errorf("check email existence: %w", err)
	}

	return otel.ReportSuccess(span, &apimodels.EmailExistsNoContent{}), nil
}
