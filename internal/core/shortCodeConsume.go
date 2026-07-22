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

var (
	// ErrShortCodeConsumeInvalid is returned by [ShortCodeConsume.Exec] when the
	// supplied code does not match the stored hash for the (Usage, Target) pair.
	// Callers map it to a 4xx response.
	ErrShortCodeConsumeInvalid = errors.New("invalid short code")
	// ErrShortCodeConsumeExpired is returned by [ShortCodeConsume.Exec] when the
	// stored code's expiry has already passed. The SQL select filters expired rows
	// too; this check covers clock skew between the database and the service.
	ErrShortCodeConsumeExpired = errors.New("short code expired")
)

// ShortCodeConsumeDaoSelect loads the stored short code for a (Usage, Target)
// pair so the submitted code can be verified against it.
type ShortCodeConsumeDaoSelect interface {
	Exec(ctx context.Context, request *dao.ShortCodeSelectRequest) (*dao.ShortCode, error)
}

// ShortCodeConsumeDaoDelete retires a short code once it has been redeemed, so it
// cannot be used a second time.
type ShortCodeConsumeDaoDelete interface {
	Exec(ctx context.Context, request *dao.ShortCodeDeleteRequest) (*dao.ShortCode, error)
}

// ShortCodeConsumeRequest identifies the code to redeem: the flow it was issued
// for, the subject it was bound to, and the plaintext code the user supplied.
type ShortCodeConsumeRequest struct {
	Usage  string `validate:"required,usage"`
	Target string `validate:"required,max=1024"`
	Code   string `validate:"required,max=1024"`
}

// ShortCodeConsume verifies a user-submitted short code against the stored hash
// and, on success, retires it so it cannot be redeemed twice. It is the
// counterpart to [ShortCodeCreate].
type ShortCodeConsume struct {
	daoSelect ShortCodeConsumeDaoSelect
	daoDelete ShortCodeConsumeDaoDelete
}

// NewShortCodeConsume wires the consume service to the DAOs that read and delete
// stored codes.
func NewShortCodeConsume(
	daoSelect ShortCodeConsumeDaoSelect,
	daoDelete ShortCodeConsumeDaoDelete,
) *ShortCodeConsume {
	return &ShortCodeConsume{
		daoSelect: daoSelect,
		daoDelete: daoDelete,
	}
}

// Exec verifies the submitted code and, on success, deletes it and returns the
// consumed [ShortCode]. It returns [ErrShortCodeConsumeInvalid] when the code does
// not match the stored hash and [ErrShortCodeConsumeExpired] when it has lapsed.
func (service *ShortCodeConsume) Exec(
	ctx context.Context, request *ShortCodeConsumeRequest,
) (*ShortCode, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ShortCodeConsume")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	entity, err := service.daoSelect.Exec(ctx, &dao.ShortCodeSelectRequest{
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

	_, err = service.daoDelete.Exec(ctx, &dao.ShortCodeDeleteRequest{
		ID:      entity.ID,
		Now:     time.Now(),
		Comment: dao.ShortCodeDeleteConsumed,
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	return otel.ReportSuccess(span, &ShortCode{
		ID:        entity.ID,
		Usage:     entity.Usage,
		Target:    entity.Target,
		Data:      entity.Data,
		CreatedAt: entity.CreatedAt,
		ExpiresAt: entity.ExpiresAt,
		PlainCode: request.Code,
	}), nil
}
