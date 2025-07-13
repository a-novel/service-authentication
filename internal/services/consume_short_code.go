package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/models"
)

var ErrInvalidShortCode = errors.New("invalid short code")

// ConsumeShortCodeSource is the source used to perform the ConsumeShortCodeService.ConsumeShortCode action.
//
// You may build one using the NewConsumeShortCodeSource function.
type ConsumeShortCodeSource interface {
	SelectShortCodeByParams(ctx context.Context, data dao.SelectShortCodeByParamsData) (*dao.ShortCodeEntity, error)
	DeleteShortCode(ctx context.Context, data dao.DeleteShortCodeData) (*dao.ShortCodeEntity, error)
}

func NewConsumeShortCodeSource(
	selectShortCode *dao.SelectShortCodeByParamsRepository,
	deleteShortCode *dao.DeleteShortCodeRepository,
) ConsumeShortCodeSource {
	return &struct {
		*dao.SelectShortCodeByParamsRepository
		*dao.DeleteShortCodeRepository
	}{
		SelectShortCodeByParamsRepository: selectShortCode,
		DeleteShortCodeRepository:         deleteShortCode,
	}
}

// ConsumeShortCodeRequest is the input used to perform the ConsumeShortCodeService.ConsumeShortCode action.
type ConsumeShortCodeRequest struct {
	// Information about the resource the short code grants access to.
	Usage models.ShortCodeUsage
	// The target that is allowed to access the resource. Only this target can retrieve the short code.
	Target string
	// The encrypted code. A clear version of this code is sent to the target.
	Code string
}

// ConsumeShortCodeService is the service used to perform the ConsumeShortCodeService.ConsumeShortCode action.
//
// You may create one using the NewConsumeShortCodeService function.
type ConsumeShortCodeService struct {
	source ConsumeShortCodeSource
}

func NewConsumeShortCodeService(source ConsumeShortCodeSource) *ConsumeShortCodeService {
	return &ConsumeShortCodeService{source: source}
}

// ConsumeShortCode validates a plain code against an encrypted short code, and expires the short code if the
// validation is successful.
//
// The plain code may be sent unencrypted, as it is only usable once. The encrypted code is stored in the database
// and is used to validate the plain code.
//
// Once used with this action, a short code expires and is no longer valid.
func (service *ConsumeShortCodeService) ConsumeShortCode(
	ctx context.Context, request ConsumeShortCodeRequest,
) (*models.ShortCode, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ConsumeShortCode")
	defer span.End()

	span.SetAttributes(
		attribute.String("request.target", request.Target),
		attribute.String("request.usage", request.Usage.String()),
	)

	entity, err := service.source.SelectShortCodeByParams(ctx, dao.SelectShortCodeByParamsData{
		Target: request.Target,
		Usage:  request.Usage,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("retrieve short code: %w", err))
	}

	span.SetAttributes(attribute.String("shortCode.id", entity.ID.String()))

	// Compare the encrypted code with the plain code of the request.
	err = lib.CompareScrypt(request.Code, entity.Code)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(
			fmt.Errorf("compare short code: %w", err),
			ErrInvalidShortCode,
		))
	}

	// Delete the short code from the database. It has been consumed.
	_, err = service.source.DeleteShortCode(ctx, dao.DeleteShortCodeData{
		ID:      entity.ID,
		Now:     time.Now(),
		Comment: dao.DeleteCommentConsumed,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("delete short code: %w", err))
	}

	return otel.ReportSuccess(span, &models.ShortCode{
		ID:        entity.ID,
		Usage:     entity.Usage,
		Target:    entity.Target,
		Data:      entity.Data,
		CreatedAt: entity.CreatedAt,
		ExpiresAt: entity.ExpiresAt,
		PlainCode: request.Code,
	}), nil
}
