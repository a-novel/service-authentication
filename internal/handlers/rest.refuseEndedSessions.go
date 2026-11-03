package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/handlers/middlewares"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

// RestRefuseEndedSessionsService checks that the caller's session still lasts: the core operation
// GET /v2/session delegates to.
type RestRefuseEndedSessionsService interface {
	Exec(ctx context.Context, request *core.ClaimsGetRequest) (*session.Claims, error)
}

// RestRefuseEndedSessions refuses a caller whose session was signed out or has ended, while its
// access token has not expired yet. A signed-out session then manages no device of its account. It
// answers as GET /v2/session does, 401 with the invalid_token challenge, so the account platform
// treats the caller as signed out. Mount [RestRefuseEndedSessions.Middleware] after the auth
// middleware, which puts the claims on the context.
type RestRefuseEndedSessions struct {
	service RestRefuseEndedSessionsService
	logger  logging.Log
}

func NewRestRefuseEndedSessions(service RestRefuseEndedSessionsService, logger logging.Log) *RestRefuseEndedSessions {
	return &RestRefuseEndedSessions{service: service, logger: logger}
}

// Middleware serves next to a caller whose session lasts, and refuses any other.
func (handler *RestRefuseEndedSessions) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if handler.refuse(w, r) {
			return
		}

		next.ServeHTTP(w, r)
	})
}

// refuse answers r and reports true when its caller's session no longer lasts.
func (handler *RestRefuseEndedSessions) refuse(w http.ResponseWriter, r *http.Request) bool {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.RefuseEndedSessions")
	defer span.End()

	claims, err := session.MustGetClaimsContext(ctx)
	if err == nil {
		_, err = handler.service.Exec(ctx, &core.ClaimsGetRequest{Claims: claims})
	}

	if err == nil {
		return false
	}

	if errors.Is(err, core.ErrClaimsGetRevoked) {
		w.Header().Set("WWW-Authenticate", middlewares.BearerChallengeInvalidToken)
	}

	httpf.HandleError(ctx, handler.logger, w, span, claimsGetErrors, err)

	return true
}
