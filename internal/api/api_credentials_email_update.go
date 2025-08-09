package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models/api"
)

type UpdateEmailService interface {
	UpdateEmail(ctx context.Context, request services.UpdateEmailRequest) (*services.UpdateEmailResponse, error)
}

func (api *API) UpdateEmail(ctx context.Context, req *apimodels.UpdateEmailForm) (apimodels.UpdateEmailRes, error) {
	ctx, span := otel.Tracer().Start(ctx, "api.UpdateEmail")
	defer span.End()

	res, err := api.UpdateEmailService.UpdateEmail(ctx, services.UpdateEmailRequest{
		UserID:    uuid.UUID(req.GetUserID()),
		ShortCode: string(req.GetShortCode()),
	})

	switch {
	case errors.Is(err, dao.ErrCredentialsNotFound):
		_ = otel.ReportError(span, err)

		return &apimodels.NotFoundError{Error: "user not found"}, nil
	case errors.Is(err, dao.ErrShortCodeNotFound), errors.Is(err, services.ErrInvalidShortCode):
		_ = otel.ReportError(span, err)

		return &apimodels.ForbiddenError{Error: "invalid short code"}, nil
	case errors.Is(err, dao.ErrCredentialsAlreadyExists):
		_ = otel.ReportError(span, err)

		return &apimodels.ConflictError{Error: "email already taken"}, nil
	case err != nil:
		_ = otel.ReportError(span, err)

		return nil, fmt.Errorf("update email: %w", err)
	}

	return otel.ReportSuccess(span, &apimodels.NewEmail{Email: apimodels.Email(res.NewEmail)}), nil
}
