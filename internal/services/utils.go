package services

import (
	"context"
	"fmt"

	jkModels "github.com/a-novel/service-json-keys/models"
	jkPkg "github.com/a-novel/service-json-keys/pkg"

	"github.com/a-novel/service-authentication/models"
)

type RefreshTokenClaimsVerifier struct {
	verifier *jkPkg.ClaimsVerifier[models.RefreshTokenClaims]
}

func (verifier *RefreshTokenClaimsVerifier) VerifyRefreshTokenClaims(
	ctx context.Context, usage jkModels.KeyUsage, accessToken string, options *jkPkg.VerifyClaimsOptions,
) (*models.RefreshTokenClaims, error) {
	claims, err := verifier.verifier.VerifyClaims(ctx, usage, accessToken, options)
	if err != nil {
		return nil, fmt.Errorf("verify refresh token claims: %w", err)
	}

	return claims, nil
}
