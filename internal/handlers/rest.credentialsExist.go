package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/core"
)

type CredentialsExistService interface {
	Exec(ctx context.Context, request *core.CredentialsExistRequest) (bool, error)
}

type CredentialsExistRequest struct {
	Email string `schema:"email"`
}

type CredentialsExist struct {
	service CredentialsExistService
	logger  logging.Log
}

func NewCredentialsExist(service CredentialsExistService, logger logging.Log) *CredentialsExist {
	return &CredentialsExist{service: service, logger: logger}
}

func (handler *CredentialsExist) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.CredentialsExist")
	defer span.End()

	var request CredentialsExistRequest

	err := muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	ok, err := handler.service.Exec(ctx, &core.CredentialsExistRequest{Email: request.Email})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
			core.ErrInvalidRequest: http.StatusUnprocessableEntity,
		}, err)

		return
	}

	// The lookup ran, so the span succeeds even where a missing email returns 404 below.
	defer otel.ReportSuccess(span, ok)

	if !ok {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
}
