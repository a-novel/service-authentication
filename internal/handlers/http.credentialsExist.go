package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel/golib/httpf"
	"github.com/a-novel/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/services"
)

type CredentialsExistService interface {
	Exec(ctx context.Context, request *services.CredentialsExistRequest) (bool, error)
}

type CredentialsExistRequest struct {
	Email string `schema:"email"`
}

type CredentialsExist struct {
	service CredentialsExistService
}

func NewCredentialsExist(service CredentialsExistService) *CredentialsExist {
	return &CredentialsExist{service: service}
}

func (handler *CredentialsExist) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "handler.CredentialsExist")
	defer span.End()

	var request CredentialsExistRequest

	err := muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	ok, err := handler.service.Exec(ctx, &services.CredentialsExistRequest{Email: request.Email})
	if err != nil {
		httpf.HandleError(ctx, w, span, httpf.ErrMap{
			services.ErrInvalidRequest: http.StatusUnprocessableEntity,
		}, err)

		return
	}

	// The status is indicative, the query is considered successful from here.
	defer otel.ReportSuccess(span, ok)

	if !ok {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
}
