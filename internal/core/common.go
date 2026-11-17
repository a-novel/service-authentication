package core

import (
	"errors"

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
	// ErrShortCodeConsumeInvalid is returned when a submitted short code does not match the
	// stored hash, or was retired or expired between its check and its redemption.
	ErrShortCodeConsumeInvalid = errors.New("invalid short code")
	// ErrWaitlistUnavailable is returned when the invitation list cannot acknowledge a request.
	ErrWaitlistUnavailable = dao.ErrWaitlistUnavailable
	// ErrWaitlistBusy is returned when the invitation list cannot admit more work.
	ErrWaitlistBusy = dao.ErrWaitlistBusy
)
