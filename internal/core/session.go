package core

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/transaction"

	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

// VisitorFieldMaxLength bounds each text a session records from its [Visitor], in characters. A
// longer text keeps its start.
const VisitorFieldMaxLength = 512

// SessionLoggedOutKept is how many logged-out sessions an account keeps, the ones active last. Each
// login and account creation deletes the older ones, which [SessionList] never shows: it returns
// [SessionListLimit] sessions at most. The same logins sign out the active sessions past
// [SessionListLimit], so the list shows every active session.
const SessionLoggedOutKept = 100

// SessionTrimBatch bounds the sessions one login signs out past [SessionListLimit], and those it
// deletes past [SessionLoggedOutKept], so each statement stays short. The next login, or the purge,
// handles the rest.
const SessionTrimBatch = 1000

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

// sessionConsentReader reads an account's location consent, and holds it against a consent update
// until the transaction ends. The consent DAO of each calling operation matches it.
type sessionConsentReader interface {
	Exec(ctx context.Context, request *dao.CredentialsLocationConsentRequest) (bool, error)
}

// sessionUpserter records an access token on its session's row. The session DAO of each calling
// operation matches it.
type sessionUpserter interface {
	Exec(ctx context.Context, request *dao.SessionUpsertRequest) error
}

// sessionRecorder records the access tokens the account platform receives on their session's row.
// Each operation that records one builds it from its own dependencies.
type sessionRecorder struct {
	transactor transaction.Transactor
	daoConsent sessionConsentReader
	daoUpsert  sessionUpserter
}

// record records an access token the account platform receives for visitor, on the row of the
// session claims belongs to, for the account userID names. consent is the account's location consent
// as the caller read it. The row records the location only under consent, and erases it otherwise.
//
// Without consent, the row is written alone: a stale "off" only records no location. With consent,
// one transaction reads the consent again and writes the row. The read waits for a consent update in
// flight, and a later consent update waits for the write, so turning the consent off erases every
// location the account's rows hold.
func (recorder sessionRecorder) record(
	ctx context.Context, userID uuid.UUID, consent bool, claims *session.RefreshClaims, visitor Visitor,
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
		UserID:    userID,
		UserAgent: clipVisitorField(visitor.UserAgent),
		Now:       time.Now(),
		ExpiresAt: time.Unix(claims.SessionExpiry(), 0),
	}

	if !consent {
		err = recorder.daoUpsert.Exec(ctx, request)
		if err != nil {
			return otel.ReportError(span, fmt.Errorf("write session: %w", err))
		}

		return nil
	}

	err = recorder.transactor.WithinTx(ctx, func(ctx context.Context) error {
		consent, err := recorder.daoConsent.Exec(ctx, &dao.CredentialsLocationConsentRequest{ID: userID})
		if err != nil {
			return fmt.Errorf("read location consent: %w", err)
		}

		if consent {
			request.IPAddress = visitor.IPAddress
			request.Region = clipVisitorField(visitor.Region)
			request.City = clipVisitorField(visitor.City)
		}

		err = recorder.daoUpsert.Exec(ctx, request)
		if err != nil {
			return fmt.Errorf("write session: %w", err)
		}

		return nil
	})
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("record session: %w", err))
	}

	return nil
}

// sessionOldestRevoker signs out an account's active sessions past a number. The DAO of each calling
// operation matches it.
type sessionOldestRevoker interface {
	Exec(ctx context.Context, request *dao.SessionRevokeOldestRequest) error
}

// sessionTrimmer deletes an account's logged-out sessions past a number. The DAO of each calling
// operation matches it.
type sessionTrimmer interface {
	Exec(ctx context.Context, request *dao.SessionTrimRequest) error
}

// sessionBounds bounds the rows of an account once a login opens a session. Each operation that
// opens one builds it from its own dependencies.
type sessionBounds struct {
	daoRevokeOldest sessionOldestRevoker
	daoTrim         sessionTrimmer
	// refreshTokenLeeway is the clock-skew tolerance the verifier grants an expired refresh token.
	refreshTokenLeeway time.Duration
}

// bound keeps the account's [SessionListLimit] active sessions active last and signs out the others,
// then deletes its logged-out sessions past the [SessionLoggedOutKept] active last. Each step handles
// [SessionTrimBatch] sessions at most. It runs after the login's own writes, and a failure fails no
// login.
func (bounds sessionBounds) bound(ctx context.Context, userID uuid.UUID) {
	ctx, span := otel.Tracer().Start(ctx, "core.boundSessions")
	defer span.End()

	revokeRequest := &dao.SessionRevokeOldestRequest{
		UserID: userID,
		Keep:   SessionListLimit,
		Limit:  SessionTrimBatch,
		Leeway: bounds.refreshTokenLeeway,
		Now:    time.Now(),
	}

	// A failure leaves its rows to the next login and to the purge. The DAO span records it.
	_ = bounds.daoRevokeOldest.Exec(ctx, revokeRequest)
	_ = bounds.daoTrim.Exec(ctx, &dao.SessionTrimRequest{
		UserID: userID, Keep: SessionLoggedOutKept, Limit: SessionTrimBatch,
	})
}

// clipVisitorField returns the first [VisitorFieldMaxLength] characters of value. It drops the bytes
// that are no UTF-8 first, since PostgreSQL refuses them in text.
func clipVisitorField(value string) string {
	characters := []rune(strings.ToValidUTF8(value, ""))

	return string(characters[:min(len(characters), VisitorFieldMaxLength)])
}
