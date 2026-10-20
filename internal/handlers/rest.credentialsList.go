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

// RestCredentialsListService is the core operation GET /v2/credentials/all delegates to.
type RestCredentialsListService interface {
	Exec(ctx context.Context, request *core.CredentialsListRequest) ([]*core.Credentials, error)
}

// credentialsListErrors maps a refused account listing to its status.
var credentialsListErrors = httpf.ErrMap{core.ErrInvalidRequest: http.StatusUnprocessableEntity}

// RestCredentialsListRequest holds the query parameters of GET /v2/credentials/all.
type RestCredentialsListRequest struct {
	Limit  int      `schema:"limit"`
	Offset int      `schema:"offset"`
	Roles  []string `schema:"roles"`
}

// RestCredentialsList serves GET /v2/credentials/all, which lists accounts.
type RestCredentialsList struct {
	service RestCredentialsListService
	logger  logging.Log
}

func NewRestCredentialsList(service RestCredentialsListService, logger logging.Log) *RestCredentialsList {
	return &RestCredentialsList{service: service, logger: logger}
}

func (handler *RestCredentialsList) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.CredentialsList")
	defer span.End()

	var request RestCredentialsListRequest

	err := muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.CredentialsListRequest{
		Limit:  request.Limit,
		Offset: request.Offset,
		Roles:  request.Roles,
	}

	res, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, credentialsListErrors, err)

		return
	}

	httpf.SendJSONStatus(ctx, w, span, http.StatusOK, lo.Map(res, loadCredentialsMap))
}
