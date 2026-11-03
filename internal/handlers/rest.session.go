package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/a-novel/service-authentication/v2/internal/core"
)

// The headers the account platform sets to describe its visitor, beside the visitor IP that
// REST_CLIENT_IP_HEADER names. The service trusts them as it trusts that IP: only the platform calls
// it.
const (
	// HeaderClientUserAgent carries the User-Agent of the visitor's browser.
	HeaderClientUserAgent = "X-Client-User-Agent"
	// HeaderClientRegion carries the CLDR region code of the visitor, such as FR.
	HeaderClientRegion = "X-Client-Region"
	// HeaderClientCity carries the city of the visitor.
	HeaderClientCity = "X-Client-City"
)

// Session is the JSON representation of a session, one device the caller signed in on, returned by
// GET /v2/sessions. The location fields are absent when the session recorded none.
type Session struct {
	ID uuid.UUID `json:"id"`

	// Current is whether the session is the caller's own.
	Current bool `json:"current"`
	// LoggedOut is whether the session ended or was signed out.
	LoggedOut bool `json:"loggedOut"`

	UserAgent string `json:"userAgent"`
	IPAddress string `json:"ipAddress,omitempty"`
	Region    string `json:"region,omitempty"`
	City      string `json:"city,omitempty"`

	CreatedAt    time.Time `json:"createdAt"`
	LastActiveAt time.Time `json:"lastActiveAt"`
	ExpiresAt    time.Time `json:"expiresAt"`
}

func loadSession(s *core.Session) Session {
	return Session{
		ID:           s.ID,
		Current:      s.Current,
		LoggedOut:    s.LoggedOut,
		UserAgent:    s.UserAgent,
		IPAddress:    s.IPAddress,
		Region:       s.Region,
		City:         s.City,
		CreatedAt:    s.CreatedAt,
		LastActiveAt: s.LastActiveAt,
		ExpiresAt:    s.ExpiresAt,
	}
}

// loadVisitor reads the visitor the account platform describes on r. The IP comes from the router,
// which reads the header REST_CLIENT_IP_HEADER names, or takes the peer's address without it.
func loadVisitor(r *http.Request) core.Visitor {
	return core.Visitor{
		UserAgent: r.Header.Get(HeaderClientUserAgent),
		IPAddress: middleware.GetClientIP(r.Context()),
		Region:    r.Header.Get(HeaderClientRegion),
		City:      r.Header.Get(HeaderClientCity),
	}
}
