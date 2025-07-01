package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/getsentry/sentry-go"

	"github.com/a-novel/service-authentication/api/codegen"
	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
)

type RegisterService interface {
	Register(ctx context.Context, request services.RegisterRequest) (*models.Token, error)
}

func (api *API) Register(ctx context.Context, req *codegen.RegisterForm) (codegen.RegisterRes, error) {
	span := sentry.StartSpan(ctx, "API.Register")
	defer span.Finish()

	span.SetData("request.email", req.GetEmail())

	token, err := api.RegisterService.Register(span.Context(), services.RegisterRequest{
		Email:     string(req.GetEmail()),
		Password:  string(req.GetPassword()),
		ShortCode: string(req.GetShortCode()),
	})

	switch {
	case errors.Is(err, dao.ErrCredentialsAlreadyExists):
		span.SetData("service.err", err.Error())

		return &codegen.ConflictError{Error: "email already taken"}, nil
	case errors.Is(err, dao.ErrShortCodeNotFound), errors.Is(err, services.ErrInvalidShortCode):
		span.SetData("service.err", err.Error())

		return &codegen.ForbiddenError{Error: "invalid short code"}, nil
	case err != nil:
		span.SetData("service.err", err.Error())

		return nil, fmt.Errorf("register user: %w", err)
	}

	return &codegen.Token{AccessToken: token.AccessToken, RefreshToken: token.RefreshToken}, nil
}
