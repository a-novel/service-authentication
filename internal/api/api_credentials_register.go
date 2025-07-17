package api

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/codes"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
	"github.com/a-novel/service-authentication/models/api"
)

type RegisterService interface {
	Register(ctx context.Context, request services.RegisterRequest) (*models.Token, error)
}

func (api *API) Register(ctx context.Context, req *apimodels.RegisterForm) (apimodels.RegisterRes, error) {
	ctx, span := otel.Tracer().Start(ctx, "api.Register")
	defer span.End()

	token, err := api.RegisterService.Register(ctx, services.RegisterRequest{
		Email:     string(req.GetEmail()),
		Password:  string(req.GetPassword()),
		ShortCode: string(req.GetShortCode()),
	})

	switch {
	case errors.Is(err, dao.ErrCredentialsAlreadyExists):
		span.RecordError(err)
		span.SetStatus(codes.Error, "")

		return &apimodels.ConflictError{Error: "email already taken"}, nil
	case errors.Is(err, dao.ErrShortCodeNotFound), errors.Is(err, services.ErrInvalidShortCode):
		span.RecordError(err)
		span.SetStatus(codes.Error, "")

		return &apimodels.ForbiddenError{Error: "invalid short code"}, nil
	case err != nil:
		span.RecordError(err)
		span.SetStatus(codes.Error, "")

		return nil, fmt.Errorf("register user: %w", err)
	}

	return otel.ReportSuccess(span, &apimodels.Token{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
	}), nil
}
