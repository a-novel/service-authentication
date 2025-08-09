package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models/api"
	"github.com/a-novel/service-authentication/pkg"
)

type UpdatePasswordService interface {
	UpdatePassword(ctx context.Context, request services.UpdatePasswordRequest) error
}

func (api *API) UpdatePassword(
	ctx context.Context, req *apimodels.UpdatePasswordForm) (apimodels.UpdatePasswordRes, error,
) {
	ctx, span := otel.Tracer().Start(ctx, "api.UpdatePassword")
	defer span.End()

	userID, err := pkg.RequireUserID(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get user id: %w", err))
	}

	err = api.UpdatePasswordService.UpdatePassword(ctx, services.UpdatePasswordRequest{
		Password:        string(req.GetPassword()),
		CurrentPassword: string(req.GetCurrentPassword()),
		UserID:          userID,
	})

	switch {
	case errors.Is(err, lib.ErrInvalidPassword):
		_ = otel.ReportError(span, err)

		return &apimodels.ForbiddenError{Error: "invalid user password"}, nil
	case err != nil:
		_ = otel.ReportError(span, err)

		return nil, fmt.Errorf("update password: %w", err)
	}

	return otel.ReportSuccess(span, &apimodels.UpdatePasswordNoContent{}), nil
}
