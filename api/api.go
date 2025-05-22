package api

import (
	"context"
	"errors"
	"github.com/a-novel-kit/middlewares/sentry"
	"net/http"

	"github.com/a-novel/service-authentication/api/codegen"
	"github.com/ogen-go/ogen/ogenerrors"
	"github.com/rs/zerolog"
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

	logger := zerolog.Ctx(ctx)

	// Return a different error if authentication failed. Also do not log error (we will still have the API log from
	// the default middleware if needed).
	var securityError *ogenerrors.SecurityError
	if ok := errors.As(err, &securityError); ok {
		logger.Warn().Err(err).Msg("authentication failed")

		switch {
		case errors.Is(err, ErrAuthentication):
			return ErrUnauthorized
		case errors.Is(err, ErrPermission):
			return ErrForbidden
		default:
			return ErrUnauthorized
		}
	}

	// Unhandled, unexpected error occurred.
	sentrymiddleware.CaptureError(ctx, err)

	return ErrInternalServerError
}
