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

// RestWaitlistJoinService records invitation requests.
type RestWaitlistJoinService interface {
	Exec(ctx context.Context, request *core.WaitlistJoinRequest) error
}

// waitlistJoinErrors maps a failed invitation-list request to its status.
var waitlistJoinErrors = httpf.ErrMap{
	core.ErrInvalidRequest:      http.StatusUnprocessableEntity,
	core.ErrWaitlistBusy:        http.StatusTooManyRequests,
	core.ErrWaitlistUnavailable: http.StatusServiceUnavailable,
}

// RestWaitlistJoinRequest is the public invitation-list payload.
type RestWaitlistJoinRequest struct {
	Email string `json:"email"`
	Lang  string `json:"lang"`
}

// waitlistRetryAfterSeconds gives the shared Google writer time to recover before another attempt.
const waitlistRetryAfterSeconds = "60"

// RestWaitlistJoin serves PUT /v2/waitlist. It never creates an account or sends an email.
type RestWaitlistJoin struct {
	service RestWaitlistJoinService
	logger  logging.Log
}

func NewRestWaitlistJoin(service RestWaitlistJoinService, logger logging.Log) *RestWaitlistJoin {
	return &RestWaitlistJoin{service: service, logger: logger}
}

// ServeHTTP returns an empty 202 for every accepted request, whether the address was new, listed or
// registered.
func (handler *RestWaitlistJoin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.WaitlistJoin")
	defer span.End()

	var request RestWaitlistJoinRequest

	err := httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	err = handler.service.Exec(ctx, &core.WaitlistJoinRequest{Email: request.Email, Lang: request.Lang})
	if err != nil {
		if errors.Is(err, core.ErrWaitlistBusy) || errors.Is(err, core.ErrWaitlistUnavailable) {
			w.Header().Set("Retry-After", waitlistRetryAfterSeconds)
		}

		httpf.HandleError(ctx, handler.logger, w, span, waitlistJoinErrors, err)

		return
	}

	w.WriteHeader(http.StatusAccepted)
}
