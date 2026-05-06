package services

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
	ErrShortCodeConsumeInvalid = errors.New("invalid short code")
	ErrShortCodeConsumeExpired = errors.New("short code expired")
)

type ShortCodeConsumeRepositorySelect interface {
	Exec(ctx context.Context, request *dao.ShortCodeSelectRequest) (*dao.ShortCode, error)
}
type ShortCodeConsumeRepositoryDelete interface {
	Exec(ctx context.Context, request *dao.ShortCodeDeleteRequest) (*dao.ShortCode, error)
}

type ShortCodeConsumeRequest struct {
	Usage  string `validate:"required,usage"`
	Target string `validate:"required,max=1024"`
	Code   string `validate:"required,max=1024"`
}

type ShortCodeConsume struct {
	repositorySelect ShortCodeConsumeRepositorySelect
	repositoryDelete ShortCodeConsumeRepositoryDelete
}

func NewShortCodeConsume(
	repositorySelect ShortCodeConsumeRepositorySelect,
	repositoryDelete ShortCodeConsumeRepositoryDelete,
) *ShortCodeConsume {
	return &ShortCodeConsume{
		repositorySelect: repositorySelect,
		repositoryDelete: repositoryDelete,
	}
}

func (service *ShortCodeConsume) Exec(
	ctx context.Context, request *ShortCodeConsumeRequest,
) (*ShortCode, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ShortCodeConsume")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, errors.Join(err, ErrInvalidRequest)
	}

	entity, err := service.repositorySelect.Exec(ctx, &dao.ShortCodeSelectRequest{
		Target: request.Target,
		Usage:  request.Usage,
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	span.SetAttributes(attribute.String("shortCode.id", entity.ID.String()))

	// Explicit expiration check as defense-in-depth (SQL already filters expired codes).
	if entity.ExpiresAt.Before(time.Now()) {
		return nil, ErrShortCodeConsumeExpired
	}

	// Compare the encrypted code with the plain code of the request.
	err = lib.CompareArgon2(request.Code, entity.Code)
	if err != nil {
		joined := errors.Join(
			fmt.Errorf("compare short code: %w", err),
			ErrShortCodeConsumeInvalid,
		)
		if errors.Is(err, lib.ErrInvalidPassword) {
			// Wrong code is the expected user-facing outcome (mistyped code, stale request).
			return nil, joined
		}
		// Other CompareArgon2 errors (lib.ErrInvalidHash, lib.ErrIncompatibleVersion) mean
		// the stored short-code hash itself is malformed — operational failure worth reporting.
		return nil, otel.ReportError(span, joined)
	}

	// Delete the short code from the database. It has been consumed.
	_, err = service.repositoryDelete.Exec(ctx, &dao.ShortCodeDeleteRequest{
		ID:      entity.ID,
		Now:     time.Now(),
		Comment: "key consumed",
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
