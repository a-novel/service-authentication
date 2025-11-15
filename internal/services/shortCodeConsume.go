package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/lib"
)

var ErrShortCodeConsumeInvalid = errors.New("invalid shortcode")

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
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	entity, err := service.repositorySelect.Exec(ctx, &dao.ShortCodeSelectRequest{
		Target: request.Target,
		Usage:  request.Usage,
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	span.SetAttributes(attribute.String("shortCode.id", entity.ID.String()))

	// Compare the encrypted code with the plain code of the request.
	err = lib.CompareScrypt(request.Code, entity.Code)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(
			fmt.Errorf("compare short code: %w", err),
			ErrShortCodeConsumeInvalid,
		))
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
