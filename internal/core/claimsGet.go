package core

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

// ErrClaimsGetRevoked is returned by [ClaimsGet.Exec] when the access token's refresh token or
// session was revoked.
var ErrClaimsGetRevoked = errors.New("session revoked")

// ClaimsGetDao reports whether the access token's refresh token or session was revoked.
type ClaimsGetDao interface {
	Exec(ctx context.Context, request *dao.RefreshTokenRevocationExistRequest) (bool, error)
}

// ClaimsGetRequest carries the caller's verified access token claims.
type ClaimsGetRequest struct {
	Claims *session.Claims `validate:"required"`
}

// ClaimsGet returns the claims of an access token while its session lasts. An anonymous token
// belongs to no session, so its claims always return.
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

	revocationRequest := &dao.RefreshTokenRevocationExistRequest{
		JTIs: []string{claims.RefreshTokenID, claims.Session()},
	}

	revoked, err := service.dao.Exec(ctx, revocationRequest)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("check session revocation: %w", err))
	}

	if revoked {
		return nil, otel.ReportError(span, ErrClaimsGetRevoked)
	}

	return claims, nil
}
