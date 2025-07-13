package api

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/codes"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
	"github.com/a-novel/service-authentication/models/api"
)

type LoginService interface {
	Login(ctx context.Context, request services.LoginRequest) (*models.Token, error)
}

func (api *API) CreateSession(ctx context.Context, req *apimodels.LoginForm) (apimodels.CreateSessionRes, error) {
	ctx, span := otel.Tracer().Start(ctx, "api.CreateSession")
	defer span.End()

	token, err := api.LoginService.Login(ctx, services.LoginRequest{
		Email:    string(req.GetEmail()),
		Password: string(req.GetPassword()),
	})

	switch {
	case errors.Is(err, dao.ErrCredentialsNotFound):
		span.RecordError(err)
		span.SetStatus(codes.Error, "")

		return &apimodels.NotFoundError{Error: "user not found"}, nil
	case errors.Is(err, lib.ErrInvalidPassword):
		span.RecordError(err)
		span.SetStatus(codes.Error, "")

		return &apimodels.ForbiddenError{Error: "invalid user password"}, nil
	case err != nil:
		span.RecordError(err)
		span.SetStatus(codes.Error, "")

		return nil, fmt.Errorf("login user: %w", err)
	}

	return otel.ReportSuccess(span, &apimodels.Token{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
	}), nil
}
