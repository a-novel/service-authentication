package handlers

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/core"
)

// RestCredentialsGetService is the core operation GET /v2/credentials delegates to.
type RestCredentialsGetService interface {
	Exec(ctx context.Context, request *core.CredentialsGetRequest) (*core.Credentials, error)
}

// RestCredentialsGetRequest holds the query parameters of GET /v2/credentials.
type RestCredentialsGetRequest struct {
	ID uuid.UUID `schema:"id"`
}

// RestCredentialsGet serves GET /v2/credentials, which returns an account.
type RestCredentialsGet struct {
	service RestCredentialsGetService
	logger  logging.Log
}

func NewRestCredentialsGet(service RestCredentialsGetService, logger logging.Log) *RestCredentialsGet {
	return &RestCredentialsGet{service: service, logger: logger}
}

func (handler *RestCredentialsGet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.CredentialsGet")
	defer span.End()

	var request RestCredentialsGetRequest

	err := muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	res, err := handler.service.Exec(ctx, &core.CredentialsGetRequest{ID: request.ID})
	if err != nil {
		errMap := httpf.ErrMap{
			core.ErrCredentialsGetNotFound: http.StatusNotFound,
			core.ErrInvalidRequest:         http.StatusUnprocessableEntity,
		}
		httpf.HandleError(ctx, handler.logger, w, span, errMap, err)

		return
	}

	httpf.SendJSONStatus(ctx, w, span, http.StatusOK, loadCredentials(res))
}
