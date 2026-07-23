package handlers

import (
	"context"
	"net/http"

	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/core"
)

type CredentialsListService interface {
	Exec(ctx context.Context, request *core.CredentialsListRequest) ([]*core.Credentials, error)
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
	ctx, span := otel.Tracer().Start(r.Context(), "rest.CredentialsList")
	defer span.End()

	var request CredentialsListRequest

	err := muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	res, err := handler.service.Exec(ctx, &core.CredentialsListRequest{
		Limit:  request.Limit,
		Offset: request.Offset,
		Roles:  request.Roles,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
			core.ErrInvalidRequest: http.StatusUnprocessableEntity,
		}, err)

		return
	}

	httpf.SendJSONStatus(ctx, w, span, http.StatusOK, lo.Map(res, loadCredentialsMap))
}
