package api

import (
	"context"
	"errors"
	"github.com/getsentry/sentry-go"
	"net/http"

	"github.com/a-novel/service-authentication/api/codegen"
	"github.com/ogen-go/ogen/ogenerrors"
)

var ErrUnauthorized = &codegen.UnexpectedErrorStatusCode{
	StatusCode: http.StatusUnauthorized,
	Response:   codegen.UnexpectedError{Error: "Unauthorized"},
}

var ErrForbidden = &codegen.UnexpectedErrorStatusCode{
	StatusCode: http.StatusForbidden,
	Response:   codegen.UnexpectedError{Error: "Forbidden"},
}

var ErrInternalServerError = &codegen.UnexpectedErrorStatusCode{
	StatusCode: http.StatusInternalServerError,
	Response:   codegen.UnexpectedError{Error: "internal server error"},
}

type API struct {
	LoginService               LoginService
	LoginAnonService           LoginAnonService
	ConsumeRefreshTokenService ConsumeRefreshTokenService
	IssueRefreshTokenService   IssueRefreshTokenService

	SelectKeyService  SelectKeyService
	SearchKeysService SearchKeysService

	RequestEmailUpdateService   RequestEmailUpdateService
	RequestPasswordResetService RequestPasswordResetService
	RequestRegisterService      RequestRegisterService

	RegisterService       RegisterService
	EmailExistsService    EmailExistsService
	UpdateEmailService    UpdateEmailService
	UpdatePasswordService UpdatePasswordService
	UpdateRoleService     UpdateRoleService

	ListUsersService ListUsersService
	GetUserService   GetUserService

	codegen.UnimplementedHandler
}

func (api *API) NewError(ctx context.Context, err error) *codegen.UnexpectedErrorStatusCode {
	// no-op
	if err == nil {
		return nil
	}
	
	logger := sentry.NewLogger(ctx)
	logger.Errorf(ctx, "security error: %v", err)

	// Return a different error if authentication failed. Also do not log error (we will still have the API log from
	// the default middleware if needed).
	var securityError *ogenerrors.SecurityError
	if ok := errors.As(err, &securityError); ok {
		switch {
		case errors.Is(err, ErrAuthentication):
			return ErrUnauthorized
		case errors.Is(err, ErrPermission):
			return ErrForbidden
		default:
			return ErrUnauthorized
		}
	}

	return ErrInternalServerError
}
