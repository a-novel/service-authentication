package handlers

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/dao"
)

type CredentialsGetService interface {
	Exec(ctx context.Context, request *core.CredentialsGetRequest) (*core.Credentials, error)
}

type CredentialsGetRequest struct {
	ID uuid.UUID `schema:"id"`
}

type CredentialsGet struct {
	service CredentialsGetService
	logger  logging.Log
}

func NewCredentialsGet(service CredentialsGetService, logger logging.Log) *CredentialsGet {
	return &CredentialsGet{service: service, logger: logger}
}

func (handler *CredentialsGet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.CredentialsGet")
	defer span.End()

	var request CredentialsGetRequest

	err := muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	res, err := handler.service.Exec(ctx, &core.CredentialsGetRequest{
		ID: request.ID,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
			dao.ErrCredentialsSelectNotFound: http.StatusNotFound,
			core.ErrInvalidRequest:           http.StatusUnprocessableEntity,
		}, err)

		return
	}

	httpf.SendJSON(ctx, w, span, loadCredentials(res))
}
