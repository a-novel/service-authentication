package core

import (
	"errors"

	"github.com/a-novel/service-authentication/v2/internal/dao"
)

var (
	// ErrCredentialsCreateAlreadyExists is returned when registration targets an existing email address.
	ErrCredentialsCreateAlreadyExists = dao.ErrCredentialsInsertAlreadyExists
	// ErrCredentialsGetNotFound is returned when the requested credentials do not exist.
	ErrCredentialsGetNotFound = dao.ErrCredentialsSelectNotFound
	// ErrCredentialsByEmailNotFound is returned when no credentials match an email address.
	ErrCredentialsByEmailNotFound = dao.ErrCredentialsSelectByEmailNotFound
	// ErrCredentialsUpdateEmailNotFound is returned when the credentials to update do not exist.
	ErrCredentialsUpdateEmailNotFound = dao.ErrCredentialsUpdateEmailNotFound
	// ErrCredentialsUpdateEmailAlreadyExists is returned when an email update conflicts with existing credentials.
	ErrCredentialsUpdateEmailAlreadyExists = dao.ErrCredentialsUpdateEmailAlreadyExists
	// ErrCredentialsUpdatePasswordNotFound is returned when the credentials to update do not exist.
	ErrCredentialsUpdatePasswordNotFound = dao.ErrCredentialsUpdatePasswordNotFound
	// ErrCredentialsUpdateRoleNotFound is returned when the credentials to update do not exist.
	ErrCredentialsUpdateRoleNotFound = dao.ErrCredentialsUpdateRoleNotFound
	// ErrShortCodeNotFound is returned when a matching short code does not exist.
	ErrShortCodeNotFound = dao.ErrShortCodeSelectNotFound
	// ErrShortCodeConsumeInvalid is returned when a submitted short code does not match the
	// stored hash, or was retired or expired between its check and its redemption.
	ErrShortCodeConsumeInvalid = errors.New("invalid short code")
	// ErrWaitlistUnavailable is returned when the invitation list cannot acknowledge a request.
	ErrWaitlistUnavailable = dao.ErrWaitlistUnavailable
	// ErrWaitlistBusy is returned when the invitation list cannot admit more work.
	ErrWaitlistBusy = dao.ErrWaitlistBusy
	// ErrWaitlistAlreadyJoined is returned when an invitation request already exists for the email.
	ErrWaitlistAlreadyJoined = dao.ErrWaitlistAlreadyJoined
)
