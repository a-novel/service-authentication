package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"

	"github.com/a-novel/service-authentication/internal/api/codegen"
	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/internal/services"
)

func (api *API) ResetPassword(ctx context.Context, req *codegen.ResetPasswordForm) (codegen.ResetPasswordRes, error) {
	span := sentry.StartSpan(ctx, "API.ResetPassword")
	defer span.Finish()

	span.SetData("request.userID", req.GetUserID())

	err := api.UpdatePasswordService.UpdatePassword(span.Context(), services.UpdatePasswordRequest{
		Password:  string(req.GetPassword()),
		ShortCode: string(req.GetShortCode()),
		UserID:    uuid.UUID(req.GetUserID()),
	})

	switch {
	case errors.Is(err, dao.ErrShortCodeNotFound), errors.Is(err, lib.ErrInvalidPassword):
		span.SetData("service.err", err.Error())

		return &codegen.ForbiddenError{Error: "invalid short code"}, nil
	case err != nil:
		span.SetData("service.err", err.Error())

		return nil, fmt.Errorf("reset password: %w", err)
	}

	return &codegen.ResetPasswordNoContent{}, nil
}
