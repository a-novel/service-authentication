package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/a-novel/service-authentication/api/codegen"
	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/services"
)

type UpdateEmailService interface {
	UpdateEmail(ctx context.Context, request services.UpdateEmailRequest) (*services.UpdateEmailResponse, error)
}

func (api *API) UpdateEmail(ctx context.Context, req *codegen.UpdateEmailForm) (codegen.UpdateEmailRes, error) {
	res, err := api.UpdateEmailService.UpdateEmail(ctx, services.UpdateEmailRequest{
		UserID:    uuid.UUID(req.GetUserID()),
		ShortCode: string(req.GetShortCode()),
	})

	switch {
	case errors.Is(err, dao.ErrCredentialsNotFound):
		return &codegen.NotFoundError{Error: "user not found"}, nil
	case errors.Is(err, dao.ErrShortCodeNotFound), errors.Is(err, services.ErrInvalidShortCode):
		return &codegen.ForbiddenError{Error: "invalid short code"}, nil
	case errors.Is(err, dao.ErrCredentialsAlreadyExists):
		return &codegen.ConflictError{Error: "email already taken"}, nil
	case err != nil:
		return nil, fmt.Errorf("update email: %w", err)
	}

	return lo.ToPtr(codegen.Email(res.NewEmail)), nil
}
