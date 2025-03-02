package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/ogen-go/ogen/ogenerrors"
	"github.com/rs/zerolog"
	"github.com/samber/lo"

	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/authentication/api/codegen"
)

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

		return &codegen.UnexpectedErrorStatusCode{
			StatusCode: http.StatusUnauthorized,
			Response:   codegen.UnexpectedError{Error: "Unauthorized"},
		}
	}

	// Unhandled, unexpected error occurred.
	logger.Error().Err(err).Msg("internal error")

	return &codegen.UnexpectedErrorStatusCode{
		StatusCode: http.StatusInternalServerError,
		Response:   codegen.UnexpectedError{Error: "internal server error"},
	}
}

func (api *API) jwkToModel(src *jwa.JWK) (*codegen.JWK, error) {
	rawPayload := new(codegen.JWKAdditional)
	if err := rawPayload.UnmarshalJSON(src.Payload); err != nil {
		return nil, fmt.Errorf("unmarshal payload: %w", err)
	}

	kid, err := uuid.Parse(src.KID)
	if err != nil {
		return nil, fmt.Errorf("parse kid: %w", err)
	}

	return &codegen.JWK{
		Kty:             codegen.KTY(src.KTY),
		Use:             codegen.Use(src.Use),
		KeyOps:          lo.Map(src.KeyOps, func(item jwa.KeyOp, _ int) codegen.KeyOp { return codegen.KeyOp(item) }),
		Alg:             codegen.Alg(src.Alg),
		Kid:             codegen.OptKID{Value: codegen.KID(kid), Set: true},
		AdditionalProps: *rawPayload,
	}, nil
}

func (api *API) jwksToModels(src ...*jwa.JWK) ([]codegen.JWK, error) {
	output := make([]codegen.JWK, len(src))

	for i, jwk := range src {
		model, err := api.jwkToModel(jwk)
		if err != nil {
			return nil, fmt.Errorf("convert jwk to model: %w", err)
		}

		output[i] = *model
	}

	return output, nil
}
