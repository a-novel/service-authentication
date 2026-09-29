package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/handlers"
	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"
)

const (
	waitlistMaxConcurrent  = 4
	waitlistMaxRequestSize = 8 << 10
)

// mountWaitlist bounds the optional Google path separately from ordinary authentication traffic.
func mountWaitlist(
	api chi.Router,
	withAuth serviceauthentication.PermissionsHandler,
	writer *dao.GoogleWaitlist,
	cfg config.App,
) {
	service := core.NewWaitlistJoin(dao.NewCredentialsExist(), writer)
	handler := handlers.NewRESTWaitlistJoin(service, cfg.Logger)
	withAuth(api.With(middleware.Throttle(waitlistMaxConcurrent), middleware.RequestSize(waitlistMaxRequestSize)),
		"waitlist:join").Put("/waitlist", handler.ServeHTTP)
}
