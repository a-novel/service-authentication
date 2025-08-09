package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models/api"
)

func (api *API) ResetPassword(
	ctx context.Context, req *apimodels.ResetPasswordForm,
) (apimodels.ResetPasswordRes, error) {
	ctx, span := otel.Tracer().Start(ctx, "api.ResetPassword")
	defer span.End()

	err := api.UpdatePasswordService.UpdatePassword(ctx, services.UpdatePasswordRequest{
		Password:  string(req.GetPassword()),
		ShortCode: string(req.GetShortCode()),
		UserID:    uuid.UUID(req.GetUserID()),
	})

	switch {
	case errors.Is(err, dao.ErrShortCodeNotFound), errors.Is(err, lib.ErrInvalidPassword):
		_ = otel.ReportError(span, err)

		return &apimodels.ForbiddenError{Error: "invalid short code"}, nil
	case err != nil:
		_ = otel.ReportError(span, err)

		return nil, fmt.Errorf("reset password: %w", err)
	}

	return otel.ReportSuccess(span, &apimodels.ResetPasswordNoContent{}), nil
}
