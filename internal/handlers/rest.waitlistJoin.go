package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/dao"
)

// RESTWaitlistJoinService accepts invitation requests without disclosing account existence.
type RESTWaitlistJoinService interface {
	Exec(ctx context.Context, request *core.WaitlistJoinRequest) error
}

// RESTWaitlistJoinRequest is the public invitation-list payload.
type RESTWaitlistJoinRequest struct {
	Email string `json:"email"`
	Lang  string `json:"lang"`
}

// RESTWaitlistJoin serves PUT /v2/waitlist. It never creates an account or sends an email.
type RESTWaitlistJoin struct {
	service RESTWaitlistJoinService
	logger  logging.Log
}

// NewRESTWaitlistJoin wires the transport to invitation-list logic and error reporting.
func NewRESTWaitlistJoin(service RESTWaitlistJoinService, logger logging.Log) *RESTWaitlistJoin {
	return &RESTWaitlistJoin{service: service, logger: logger}
}

// ServeHTTP returns an empty 202 for new, duplicate, and already-registered addresses.
func (handler *RESTWaitlistJoin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.WaitlistJoin")
	defer span.End()

	var request RESTWaitlistJoinRequest

	err := httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	err = handler.service.Exec(ctx, &core.WaitlistJoinRequest{Email: request.Email, Lang: request.Lang})
	if err != nil {
		if errors.Is(err, dao.ErrWaitlistBusy) || errors.Is(err, dao.ErrWaitlistUnavailable) {
			w.Header().Set("Retry-After", "60")
		}

		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
			core.ErrInvalidRequest:     http.StatusUnprocessableEntity,
			dao.ErrWaitlistBusy:        http.StatusTooManyRequests,
			dao.ErrWaitlistUnavailable: http.StatusServiceUnavailable,
		}, err)

		return
	}

	w.WriteHeader(http.StatusAccepted)
}
