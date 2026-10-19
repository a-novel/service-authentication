package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/core"
)

// CredentialsExistService is the core operation HEAD /v2/credentials delegates to.
type CredentialsExistService interface {
	Exec(ctx context.Context, request *core.CredentialsExistRequest) (bool, error)
}

// CredentialsExistRequest holds the query parameters of HEAD /v2/credentials.
type CredentialsExistRequest struct {
	Email string `schema:"email"`
}

// CredentialsExist serves HEAD /v2/credentials, which reports whether an account uses an email.
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

	if !ok {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
}
