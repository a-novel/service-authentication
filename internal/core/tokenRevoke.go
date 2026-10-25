package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

var (
	// ErrTokenRevokeInvalidRefreshToken is returned by [TokenRevoke.Exec] when the verifier rejects
	// the refresh token itself, joined onto that verifier error. Expiry is ignored, so an expired
	// refresh token revokes without error.
	ErrTokenRevokeInvalidRefreshToken = errors.New("invalid refresh token")
	// ErrTokenRevokeMismatchClaims is returned by [TokenRevoke.Exec] when the refresh token belongs
	// to another user than the caller. A caller revokes only their own sessions.
	ErrTokenRevokeMismatchClaims = errors.New("refresh token claims don't match access token")
)

// TokenRevokeDao records the revoked refresh token.
type TokenRevokeDao interface {
	Exec(ctx context.Context, request *dao.RefreshTokenRevocationInsertRequest) error
}

// TokenRevokeServiceVerifyRefreshClaims verifies the refresh token and decodes its claims. Same
// published-client method name as [TokenRefreshServiceVerifyClaims].
//
// nosemgrep: a-novel-dep-interface-method-must-be-exec
type TokenRevokeServiceVerifyRefreshClaims interface {
	VerifyClaims(ctx context.Context, req *servicejsonkeys.VerifyClaimsRequest) (*session.RefreshClaims, error)
}

// TokenRevokeRequest carries the refresh token of the session to end, and the caller it must belong
// to.
type TokenRevokeRequest struct {
	// UserID is the caller, taken from their verified access token.
	UserID       uuid.UUID
	RefreshToken string `validate:"required,max=1024"`
}

// TokenRevoke ends a session: the refresh token it revokes no longer renews the session. The
// refresh token must belong to the caller. Revoking a token already revoked or expired succeeds.
// Access tokens the session already issued stay valid until they expire.
type TokenRevoke struct {
	dao                        TokenRevokeDao
	serviceVerifyRefreshClaims TokenRevokeServiceVerifyRefreshClaims
	// refreshTokenLeeway is the clock-skew tolerance the verifier grants an expired refresh token.
	refreshTokenLeeway time.Duration
}

func NewTokenRevoke(
	dao TokenRevokeDao,
	serviceVerifyRefreshClaims TokenRevokeServiceVerifyRefreshClaims,
	refreshTokenLeeway time.Duration,
) *TokenRevoke {
	return &TokenRevoke{
		dao:                        dao,
		serviceVerifyRefreshClaims: serviceVerifyRefreshClaims,
		refreshTokenLeeway:         refreshTokenLeeway,
	}
}

func (service *TokenRevoke) Exec(ctx context.Context, request *TokenRevokeRequest) error {
	ctx, span := otel.Tracer().Start(ctx, "core.TokenRevoke")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	verifyRequest := &servicejsonkeys.VerifyClaimsRequest{
		Usage:       servicejsonkeys.KeyUsageAuthRefresh,
		AccessToken: request.RefreshToken,
		Options:     &servicejsonkeys.VerifyClaimsOptions{IgnoreExpired: true},
	}

	claims, err := service.serviceVerifyRefreshClaims.VerifyClaims(ctx, verifyRequest)
	if err != nil {
		return otel.ReportError(span, joinInvalidToken(err, ErrTokenRevokeInvalidRefreshToken))
	}

	span.SetAttributes(
		attribute.String("access_token.user_id", request.UserID.String()),
		attribute.String("refresh_token.id", claims.Jti),
		attribute.String("refresh_token.user_id", claims.UserID.String()),
	)

	if claims.UserID != request.UserID {
		return otel.ReportError(span, ErrTokenRevokeMismatchClaims)
	}

	// The verifier accepts the token until its expiry plus the leeway, so the revocation lasts as long.
	revocation := &dao.RefreshTokenRevocationInsertRequest{
		JTI:       claims.Jti,
		ExpiresAt: time.Unix(claims.Exp, 0).Add(service.refreshTokenLeeway),
		Now:       time.Now(),
	}

	err = service.dao.Exec(ctx, revocation)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("revoke refresh token: %w", err))
	}

	return nil
}
