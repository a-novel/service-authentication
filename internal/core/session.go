package core

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

// VisitorFieldMaxLength bounds each text a session records from its [Visitor], in characters. A
// longer text keeps its start.
const VisitorFieldMaxLength = 512

// Visitor is the device a request comes from, as the account platform forwards it. Every field is
// empty when unknown.
type Visitor struct {
	// UserAgent is the User-Agent of the device's browser.
	UserAgent string
	// IPAddress is the device's IP address.
	IPAddress string `validate:"omitempty,ip"`
	// Region is the CLDR region code of the device's location, such as FR.
	Region string
	// City is the city of the device's location.
	City string
}

// Session is one login on the account platform, joined by the OAuth clients that sign in through it.
// The account platform shows it as a device. Its location fields are empty unless the account
// consents to locations. A session is active until it ends or is signed out, then logged out until
// the account removes it or [SessionRetention] passes.
type Session struct {
	// ID is the JTI of the refresh token that opened the session.
	ID uuid.UUID

	// Current is whether the session is the caller's own.
	Current bool
	// LoggedOut is whether the session ended or was signed out.
	LoggedOut bool

	// UserAgent is the User-Agent of the device's browser, empty when unknown.
	UserAgent string
	// IPAddress is the device's IP address.
	IPAddress string
	// Region is the CLDR region code of the device's location, such as FR.
	Region string
	// City is the city of the device's location.
	City string

	CreatedAt time.Time
	// LastActiveAt is when the session's latest access token was issued.
	LastActiveAt time.Time
	// ExpiresAt is when the session ends.
	ExpiresAt time.Time
}

// newSession returns the session entity holds. It is current when its ID is currentID.
func newSession(entity *dao.Session, currentID string) *Session {
	return &Session{
		ID:           entity.ID,
		Current:      entity.ID.String() == currentID,
		LoggedOut:    entity.LoggedOut,
		UserAgent:    entity.UserAgent,
		IPAddress:    entity.IPAddress,
		Region:       entity.Region,
		City:         entity.City,
		CreatedAt:    entity.CreatedAt,
		LastActiveAt: entity.LastActiveAt,
		ExpiresAt:    entity.ExpiresAt,
	}
}

// sessionUpserter records an access token on its session's row. The session DAO of each calling
// operation matches it.
type sessionUpserter interface {
	Exec(ctx context.Context, request *dao.SessionUpsertRequest) error
}

// recordSession records an access token the account platform receives for visitor, on the row of the
// session claims belongs to. The row records the location only when credentials consent to it, and
// erases it otherwise.
func recordSession(
	ctx context.Context, upserter sessionUpserter, claims *session.RefreshClaims, credentials *dao.Credentials,
	visitor Visitor,
) error {
	ctx, span := otel.Tracer().Start(ctx, "core.recordSession")
	defer span.End()

	// A login learns its session only here, from the refresh token it signed.
	span.SetAttributes(attribute.String("session.id", claims.Session()))

	id, err := uuid.Parse(claims.Session())
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("parse session ID: %w", err))
	}

	request := &dao.SessionUpsertRequest{
		ID:        id,
		UserID:    credentials.ID,
		UserAgent: clipVisitorField(visitor.UserAgent),
		Now:       time.Now(),
		ExpiresAt: time.Unix(claims.SessionExpiry(), 0),
	}

	if credentials.LocationConsent {
		request.IPAddress = visitor.IPAddress
		request.Region = clipVisitorField(visitor.Region)
		request.City = clipVisitorField(visitor.City)
	}

	err = upserter.Exec(ctx, request)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("record session: %w", err))
	}

	return nil
}

// clipVisitorField returns the first [VisitorFieldMaxLength] characters of value. It drops the bytes
// that are no UTF-8 first, since PostgreSQL refuses them in text.
func clipVisitorField(value string) string {
	characters := []rune(strings.ToValidUTF8(value, ""))

	return string(characters[:min(len(characters), VisitorFieldMaxLength)])
}
