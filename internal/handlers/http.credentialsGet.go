package handlers

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/services"
)

type CredentialsGetService interface {
	Exec(ctx context.Context, request *services.CredentialsGetRequest) (*services.Credentials, error)
}

type CredentialsGetRequest struct {
	ID uuid.UUID `schema:"id"`
}

type CredentialsGet struct {
	service CredentialsGetService
}

func NewCredentialsGet(service CredentialsGetService) *CredentialsGet {
	return &CredentialsGet{service: service}
}

func (handler *CredentialsGet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "handler.CredentialsGet")
	defer span.End()

	var request CredentialsGetRequest

	err := muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	res, err := handler.service.Exec(ctx, &services.CredentialsGetRequest{
		ID: request.ID,
	})
	if err != nil {
		httpf.HandleError(ctx, w, span, httpf.ErrMap{
			dao.ErrCredentialsSelectNotFound: http.StatusNotFound,
			services.ErrInvalidRequest:       http.StatusUnprocessableEntity,
		}, err)

		return
	}

	httpf.SendJSON(ctx, w, span, loadCredentials(res))
}
