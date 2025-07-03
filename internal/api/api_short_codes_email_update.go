package api

import (
	"context"
	"fmt"

	"github.com/getsentry/sentry-go"

	"github.com/a-novel/service-authentication/internal/api/codegen"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
	"github.com/a-novel/service-authentication/pkg"
)

type RequestEmailUpdateService interface {
	RequestEmailUpdate(ctx context.Context, request services.RequestEmailUpdateRequest) (*models.ShortCode, error)
}

func (api *API) RequestEmailUpdate(
	ctx context.Context, req *codegen.RequestEmailUpdateForm,
) (codegen.RequestEmailUpdateRes, error) {
	span := sentry.StartSpan(ctx, "API.RequestEmailUpdate")
	defer span.Finish()

	span.SetData("request.email", req.GetEmail())
	span.SetData("request.lang", req.GetLang().Value)

	userID, err := pkg.RequireUserID(span.Context())
	if err != nil {
		span.SetData("request.userID.err", err.Error())

		return nil, fmt.Errorf("require user ID: %w", err)
	}

	span.SetData("request.userID", userID)

	_, err = api.RequestEmailUpdateService.RequestEmailUpdate(span.Context(), services.RequestEmailUpdateRequest{
		Email: string(req.GetEmail()),
		Lang:  models.Lang(req.GetLang().Value),
		ID:    userID,
	})
	if err != nil {
		span.SetData("service.err", err.Error())

		return nil, fmt.Errorf("request email update: %w", err)
	}

	return &codegen.RequestEmailUpdateNoContent{}, nil
}
