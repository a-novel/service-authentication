package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"

	"github.com/a-novel/service-authentication/api/codegen"
	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/services"
)

type UpdateEmailService interface {
	UpdateEmail(ctx context.Context, request services.UpdateEmailRequest) (*services.UpdateEmailResponse, error)
}

func (api *API) UpdateEmail(ctx context.Context, req *codegen.UpdateEmailForm) (codegen.UpdateEmailRes, error) {
	span := sentry.StartSpan(ctx, "API.UpdateEmail")
	defer span.Finish()

	span.SetData("request.userID", req.GetUserID())

	res, err := api.UpdateEmailService.UpdateEmail(span.Context(), services.UpdateEmailRequest{
		UserID:    uuid.UUID(req.GetUserID()),
		ShortCode: string(req.GetShortCode()),
	})

	switch {
	case errors.Is(err, dao.ErrCredentialsNotFound):
		span.SetData("service.err", err.Error())

		return &codegen.NotFoundError{Error: "user not found"}, nil
	case errors.Is(err, dao.ErrShortCodeNotFound), errors.Is(err, services.ErrInvalidShortCode):
		span.SetData("service.err", err.Error())

		return &codegen.ForbiddenError{Error: "invalid short code"}, nil
	case errors.Is(err, dao.ErrCredentialsAlreadyExists):
		span.SetData("service.err", err.Error())

		return &codegen.ConflictError{Error: "email already taken"}, nil
	case err != nil:
		span.SetData("service.err", err.Error())

		return nil, fmt.Errorf("update email: %w", err)
	}

	span.SetData("service.newEmail", res.NewEmail)

	return &codegen.NewEmail{Email: codegen.Email(res.NewEmail)}, nil
}
