package handlers

import (
	"context"
	"net/http"

	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/services"
)

type CredentialsListService interface {
	Exec(ctx context.Context, request *services.CredentialsListRequest) ([]*services.Credentials, error)
}

type CredentialsListRequest struct {
	Limit  int      `schema:"limit"`
	Offset int      `schema:"offset"`
	Roles  []string `schema:"roles"`
}

type CredentialsList struct {
	service CredentialsListService
	logger  logging.Log
}

func NewCredentialsList(service CredentialsListService, logger logging.Log) *CredentialsList {
	return &CredentialsList{service: service, logger: logger}
}

func (handler *CredentialsList) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "handler.CredentialsList")
	defer span.End()

	var request CredentialsListRequest

	err := muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	res, err := handler.service.Exec(ctx, &services.CredentialsListRequest{
		Limit:  request.Limit,
		Offset: request.Offset,
		Roles:  request.Roles,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
			services.ErrInvalidRequest: http.StatusUnprocessableEntity,
		}, err)

		return
	}

	httpf.SendJSON(ctx, w, span, lo.Map(res, loadCredentialsMap))
}
