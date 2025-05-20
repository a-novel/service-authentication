package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/a-novel/service-authentication/api/codegen"
	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/services"
)

type RegisterService interface {
	Register(ctx context.Context, request services.RegisterRequest) (string, error)
}

func (api *API) Register(ctx context.Context, req *codegen.RegisterForm) (codegen.RegisterRes, error) {
	accessToken, err := api.RegisterService.Register(ctx, services.RegisterRequest{
		Email:     string(req.GetEmail()),
		Password:  string(req.GetPassword()),
		ShortCode: string(req.GetShortCode()),
	})

	switch {
	case errors.Is(err, dao.ErrCredentialsAlreadyExists):
		return &codegen.ConflictError{Error: "email already taken"}, nil
	case errors.Is(err, dao.ErrShortCodeNotFound), errors.Is(err, services.ErrInvalidShortCode):
		return &codegen.ForbiddenError{Error: "invalid short code"}, nil
	case err != nil:
		return nil, fmt.Errorf("register user: %w", err)
	}

	return &codegen.Token{AccessToken: accessToken}, nil
}
