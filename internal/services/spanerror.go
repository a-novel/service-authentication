package services

import (
	"errors"

	"go.opentelemetry.io/otel/trace"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/lib"
)

// expectedErrors lists the user-facing sentinels this package surfaces:
// validation failures, "wrong input" outcomes, not-found, and security
// mismatches. They are the normal consequence of bad client input, not service
// failures, so they must not be recorded on the span (see [reportUnexpected]).
var expectedErrors = []error{
	ErrInvalidRequest,
	ErrShortCodeConsumeInvalid,
	ErrShortCodeConsumeExpired,
	ErrTokenRefreshInvalidAccessToken,
	ErrTokenRefreshInvalidRefreshToken,
	ErrTokenRefreshMismatchClaims,
	ErrTokenRefreshMismatchSource,
	ErrCredentialsUpdateRoleSelfUpdate,
	ErrCredentialsUpdateRoleDowngradeSuperior,
	ErrCredentialsUpdateRoleToHigher,
	lib.ErrInvalidPassword,
	dao.ErrShortCodeSelectNotFound,
	dao.ErrShortCodeDeleteNotFound,
	dao.ErrShortCodeInsertAlreadyExists,
	dao.ErrCredentialsSelectNotFound,
	dao.ErrCredentialsSelectByEmailNotFound,
	dao.ErrCredentialsInsertAlreadyExists,
	dao.ErrCredentialsUpdateEmailAlreadyExists,
	dao.ErrCredentialsUpdateEmailNotFound,
	dao.ErrCredentialsUpdatePasswordNotFound,
	dao.ErrCredentialsUpdateRoleNotFound,
}

// reportUnexpected returns err unchanged when it is (or wraps) one of the
// package's expected user-facing sentinels; otherwise it records err on span
// before returning it. Use it for a service's terminal error when that error
// could be either a client-input outcome or a genuine failure — most commonly
// the error bubbling out of a postgres.RunInTx callback, which may carry a
// sentinel raised several layers down.
func reportUnexpected(span trace.Span, err error) error {
	for _, sentinel := range expectedErrors {
		if errors.Is(err, sentinel) {
			return err
		}
	}

	return otel.ReportError(span, err)
}
