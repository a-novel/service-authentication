package core

import "github.com/a-novel/service-authentication/v2/internal/dao"

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
)
