package core

import (
	"cmp"
	"context"
	"errors"
	"math"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

// ErrClaimsGetRevoked is returned by [ClaimsGet.Exec] when the access token's refresh token or
// session was revoked, or its session ended.
var ErrClaimsGetRevoked = errors.New("session revoked")

// ClaimsGetDao reports whether the access token's refresh token or session was revoked.
type ClaimsGetDao interface {
	Exec(ctx context.Context, request *dao.RefreshTokenRevocationExistRequest) (bool, error)
}

// ClaimsGetRequest carries the caller's verified access token claims.
type ClaimsGetRequest struct {
	Claims *session.Claims `validate:"required"`
}

// ClaimsGet returns the claims of an access token while its session lasts: until its end, and
// unless it was signed out. Its end bounds a token after the purge drops the session's revocation.
// An anonymous token belongs to no session, so its claims always return.
type ClaimsGet struct {
	dao ClaimsGetDao
}

func NewClaimsGet(dao ClaimsGetDao) *ClaimsGet {
	return &ClaimsGet{dao: dao}
}

func (service *ClaimsGet) Exec(ctx context.Context, request *ClaimsGetRequest) (*session.Claims, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ClaimsGet")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	claims := request.Claims
	if claims.Session() == "" {
		return claims, nil
	}

	span.SetAttributes(
		attribute.String("access_token.refresh_token_id", claims.RefreshTokenID),
		attribute.String("session.id", claims.Session()),
	)

	err = checkSessionActive(ctx, service.dao, sessionActiveOpts{
		jtis: []string{claims.RefreshTokenID, claims.Session()},
		// An access token signed before sessions carried their end is bounded by the revocation alone.
		endsAt: cmp.Or(claims.SessionExp, math.MaxInt64),
		ended:  ErrClaimsGetRevoked,
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	return claims, nil
}
