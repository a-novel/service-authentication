package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/a-novel/service-authentication/api/codegen"
	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/internal/services"
)

type LoginService interface {
	Login(ctx context.Context, request services.LoginRequest) (string, error)
}

func (api *API) CreateSession(ctx context.Context, req *codegen.LoginForm) (codegen.CreateSessionRes, error) {
	accessToken, err := api.LoginService.Login(ctx, services.LoginRequest{
		Email:    string(req.GetEmail()),
		Password: string(req.GetPassword()),
	})

	switch {
	case errors.Is(err, dao.ErrCredentialsNotFound):
		return &codegen.NotFoundError{Error: "user not found"}, nil
	case errors.Is(err, lib.ErrInvalidPassword):
		return &codegen.ForbiddenError{Error: "invalid user password"}, nil
	case err != nil:
		return nil, fmt.Errorf("login user: %w", err)
	}

	return &codegen.Token{AccessToken: accessToken}, nil
}
