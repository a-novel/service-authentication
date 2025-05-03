package api

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/a-novel-kit/context"

	"github.com/a-novel/service-authentication/api/codegen"
	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/internal/services"
)

func (api *API) ResetPassword(ctx context.Context, req *codegen.ResetPasswordForm) (codegen.ResetPasswordRes, error) {
	logger := zerolog.Ctx(ctx)

	err := api.UpdatePasswordService.UpdatePassword(ctx, services.UpdatePasswordRequest{
		Password:  string(req.GetPassword()),
		ShortCode: string(req.GetShortCode()),
		UserID:    uuid.UUID(req.GetUserID()),
	})
	if err != nil {
		logger.Err(err).Msg("ResetPassword")
	}

	switch {
	case errors.Is(err, dao.ErrShortCodeNotFound), errors.Is(err, lib.ErrInvalidPassword):
		return &codegen.ForbiddenError{Error: "invalid short code"}, nil
	case err != nil:
		return nil, fmt.Errorf("reset password: %w", err)
	}

	return &codegen.ResetPasswordNoContent{}, nil
}
