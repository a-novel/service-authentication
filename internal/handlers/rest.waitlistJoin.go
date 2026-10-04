package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/core"
)

// RESTWaitlistJoinService records invitation requests and distinguishes membership conflicts.
type RESTWaitlistJoinService interface {
	Exec(ctx context.Context, request *core.WaitlistJoinRequest) error
}

// RESTWaitlistJoinRequest is the public invitation-list payload.
type RESTWaitlistJoinRequest struct {
	Email string `json:"email"`
	Lang  string `json:"lang"`
}

// RESTWaitlistJoinConflict exposes only the stable reason the UI needs for its warning.
type RESTWaitlistJoinConflict struct {
	// Code is account_exists or already_waitlisted; neither includes the submitted address.
	Code string `json:"code"`
}

// waitlistRetryAfterSeconds gives the shared Google writer time to recover before another attempt.
const waitlistRetryAfterSeconds = "60"

// RESTWaitlistJoin serves PUT /v2/waitlist. It never creates an account or sends an email.
type RESTWaitlistJoin struct {
	service RESTWaitlistJoinService
	logger  logging.Log
}

// NewRESTWaitlistJoin wires the transport to invitation-list logic and error reporting.
func NewRESTWaitlistJoin(service RESTWaitlistJoinService, logger logging.Log) *RESTWaitlistJoin {
	return &RESTWaitlistJoin{service: service, logger: logger}
}

// ServeHTTP returns an empty 202 for new requests and a coded 409 for membership conflicts.
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
		conflict := RESTWaitlistJoinConflict{}

		switch {
		case errors.Is(err, core.ErrWaitlistAccountExists):
			conflict.Code = "account_exists"
		case errors.Is(err, core.ErrWaitlistAlreadyJoined):
			conflict.Code = "already_waitlisted"
		}

		if conflict.Code != "" {
			httpf.SendJSONStatus(ctx, w, span, http.StatusConflict, conflict)

			return
		}

		if errors.Is(err, core.ErrWaitlistBusy) || errors.Is(err, core.ErrWaitlistUnavailable) {
			w.Header().Set("Retry-After", waitlistRetryAfterSeconds)
		}

		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
			core.ErrInvalidRequest:      http.StatusUnprocessableEntity,
			core.ErrWaitlistBusy:        http.StatusTooManyRequests,
			core.ErrWaitlistUnavailable: http.StatusServiceUnavailable,
		}, err)

		return
	}

	w.WriteHeader(http.StatusAccepted)
}
