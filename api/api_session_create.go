package api

import (
	"errors"
	"fmt"

	"github.com/a-novel-kit/context"

	"github.com/a-novel/authentication/api/codegen"
	"github.com/a-novel/authentication/internal/dao"
	"github.com/a-novel/authentication/internal/lib"
	"github.com/a-novel/authentication/internal/services"
)

func (api *API) CreateSession(ctx context.Context, req *codegen.LoginForm) (codegen.CreateSessionRes, error) {
	accessToken, err := api.LoginService.Login(ctx, services.LoginRequest{
		Email:    string(req.Email),
		Password: string(req.Password),
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
