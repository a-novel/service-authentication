package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/dao"
)

var (
	// ErrCredentialsNotFound is returned when no account matches the request.
	ErrCredentialsNotFound = dao.ErrCredentialsNotFound
	// ErrCredentialsAlreadyExists is returned when another account already holds the email.
	ErrCredentialsAlreadyExists = dao.ErrCredentialsAlreadyExists
	// ErrSessionNotFound is returned when the caller holds no session of the requested ID. Another
	// account's session answers it too.
	ErrSessionNotFound = dao.ErrSessionNotFound
	// ErrShortCodeNotFound is returned when a matching short code does not exist.
	ErrShortCodeNotFound = dao.ErrShortCodeNotFound
	// ErrShortCodeInvalid is returned when a submitted short code is missing, lapsed or does not
	// match the stored hash, or was retired between its check and its redemption.
	ErrShortCodeInvalid = errors.New("invalid short code")
	// ErrWaitlistUnavailable is returned when the invitation list cannot acknowledge a request.
	ErrWaitlistUnavailable = dao.ErrWaitlistUnavailable
	// ErrWaitlistBusy is returned when the invitation list cannot admit more work.
	ErrWaitlistBusy = dao.ErrWaitlistBusy
)

// sessionRevocationDao reports whether a refresh token or a session was signed out.
type sessionRevocationDao interface {
	Exec(ctx context.Context, request *dao.RefreshTokenRevocationExistRequest) (bool, error)
}

// sessionLastsOpts names the session [checkSessionLasts] checks.
type sessionLastsOpts struct {
	// jtis are the IDs a sign-out revokes: the refresh token, the session, or both.
	jtis []string
	// endsAt is when the session ends, as a Unix timestamp in seconds. Zero bounds nothing.
	endsAt int64
	// ended is the caller's sentinel for a session that ended or was signed out.
	ended error
}

// checkSessionLasts returns opts.ended when the session reached its end or was signed out. The end
// is checked first, so an ended session costs no lookup.
func checkSessionLasts(ctx context.Context, daoRevocation sessionRevocationDao, opts sessionLastsOpts) error {
	ctx, span := otel.Tracer().Start(ctx, "core.checkSessionLasts")
	defer span.End()

	if opts.endsAt != 0 && time.Now().Unix() >= opts.endsAt {
		return otel.ReportError(span, fmt.Errorf("session ended: %w", opts.ended))
	}

	revoked, err := daoRevocation.Exec(ctx, &dao.RefreshTokenRevocationExistRequest{JTIs: opts.jtis})
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("check session revocation: %w", err))
	}

	if revoked {
		return otel.ReportError(span, fmt.Errorf("signed out: %w", opts.ended))
	}

	return nil
}

// purgeBatches calls purge until a batch deletes fewer than batch rows, so each statement holds its
// row locks briefly. It returns the rows deleted so far, even on failure, and a later run resumes
// where it stopped.
func purgeBatches(batch int64, purge func() (int64, error)) (int64, error) {
	var total int64

	for {
		purged, err := purge()
		if err != nil {
			return total, err
		}

		total += purged

		if purged < batch {
			return total, nil
		}
	}
}
