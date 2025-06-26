package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/getsentry/sentry-go"

	"github.com/a-novel/service-authentication/api/codegen"
	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/internal/services"
)

type LoginService interface {
	Login(ctx context.Context, request services.LoginRequest) (string, error)
}

func (api *API) CreateSession(ctx context.Context, req *codegen.LoginForm) (codegen.CreateSessionRes, error) {
	span := sentry.StartSpan(ctx, "API.CreateSession")
	defer span.Finish()

	span.SetData("request.email", req.GetEmail())

	accessToken, err := api.LoginService.Login(span.Context(), services.LoginRequest{
		Email:    string(req.GetEmail()),
		Password: string(req.GetPassword()),
	})

	switch {
	case errors.Is(err, dao.ErrCredentialsNotFound):
		span.SetData("service.err", err.Error())

		return &codegen.NotFoundError{Error: "user not found"}, nil
	case errors.Is(err, lib.ErrInvalidPassword):
		span.SetData("service.err", err.Error())

		return &codegen.ForbiddenError{Error: "invalid user password"}, nil
	case err != nil:
		span.SetData("service.err", err.Error())

		return nil, fmt.Errorf("login user: %w", err)
	}

	return &codegen.Token{AccessToken: accessToken}, nil
}
