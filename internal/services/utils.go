package services

import (
	"context"
	"fmt"

	jkmodels "github.com/a-novel/service-json-keys/models"
	jkpkg "github.com/a-novel/service-json-keys/pkg"

	"github.com/a-novel/service-authentication/models"
)

type RefreshTokenClaimsVerifier struct {
	verifier *jkpkg.ClaimsVerifier[models.RefreshTokenClaims]
}

func (verifier *RefreshTokenClaimsVerifier) VerifyRefreshTokenClaims(
	ctx context.Context, usage jkmodels.KeyUsage, accessToken string, options *jkpkg.VerifyClaimsOptions,
) (*models.RefreshTokenClaims, error) {
	claims, err := verifier.verifier.VerifyClaims(ctx, usage, accessToken, options)
	if err != nil {
		return nil, fmt.Errorf("verify refresh token claims: %w", err)
	}

	return claims, nil
}
