package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/lib"
)

// ErrShortCodeConsumeExpired is returned by [ShortCodeVerify.Exec] when the
// stored code's expiry has already passed. The SQL select filters expired rows
// too; this check covers clock skew between the database and the service.
var ErrShortCodeConsumeExpired = errors.New("short code expired")

// ShortCodeVerifyDao loads the stored short code for a (Usage, Target) pair so
// the submitted code can be checked against it.
type ShortCodeVerifyDao interface {
	Exec(ctx context.Context, request *dao.ShortCodeSelectRequest) (*dao.ShortCode, error)
}

// ShortCodeVerifyRequest identifies the code to check: the flow it was issued
// for, the subject it was bound to, and the plaintext code the user supplied.
type ShortCodeVerifyRequest struct {
	Usage  string `validate:"required,usage"`
	Target string `validate:"required,max=1024"`
	Code   string `validate:"required,max=1024"`
}

// ShortCodeVerify checks a user-submitted short code against the stored hash
// without retiring it. The check runs Argon2, so call it before opening a
// transaction, then retire the code with [ShortCodeConsume] inside it.
type ShortCodeVerify struct {
	dao ShortCodeVerifyDao
}

// NewShortCodeVerify wires the verify service to the DAO that reads stored codes.
func NewShortCodeVerify(dao ShortCodeVerifyDao) *ShortCodeVerify {
	return &ShortCodeVerify{dao: dao}
}

// Exec returns the stored [ShortCode] when the submitted code matches it. It
// returns [ErrShortCodeConsumeInvalid] when the code does not match the stored
// hash and [ErrShortCodeConsumeExpired] when it has lapsed.
func (service *ShortCodeVerify) Exec(ctx context.Context, request *ShortCodeVerifyRequest) (*ShortCode, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ShortCodeVerify")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	entity, err := service.dao.Exec(ctx, &dao.ShortCodeSelectRequest{
		Target: request.Target,
		Usage:  request.Usage,
	})
	if err != nil {
		if errors.Is(err, dao.ErrShortCodeSelectNotFound) {
			// Burn an Argon2id verification so a missing code costs the same as a
			// wrong one, and the latency reveals nothing about whether a code is in
			// flight for the target.
			lib.DummyCompareArgon2(request.Code)
		}

		return nil, otel.ReportError(span, err)
	}

	span.SetAttributes(attribute.String("shortCode.id", entity.ID.String()))

	if entity.ExpiresAt.Before(time.Now()) {
		return nil, otel.ReportError(span, ErrShortCodeConsumeExpired)
	}

	err = lib.CompareArgon2(request.Code, entity.Code)
	if err != nil {
		// A mistyped or stale code yields lib.ErrInvalidPassword; a malformed stored hash
		// yields lib.ErrInvalidHash or lib.ErrIncompatibleVersion. Both surface to the
		// caller as ErrShortCodeConsumeInvalid.
		return nil, otel.ReportError(span, errors.Join(
			fmt.Errorf("compare short code: %w", err),
			ErrShortCodeConsumeInvalid,
		))
	}

	return newShortCode(entity, request.Code), nil
}
