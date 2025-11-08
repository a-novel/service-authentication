package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel/service-authentication/internal/config"
	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/lib"
)

type ShortCodeCreateRepository interface {
	Exec(ctx context.Context, request *dao.ShortCodeInsertRequest) (*dao.ShortCode, error)
}

type ShortCodeCreateRequest struct {
	Usage  string `validate:"required,usage"`
	Target string `validate:"required,max=1024"`
	Data   any
	TTL    time.Duration `validate:"required"`

	// Whe true, automatically expires any existing short code with the same target and usage.
	// Otherwise, the presence of a duplicate will trigger an error.
	Override bool
}

type ShortCodeCreate struct {
	repository ShortCodeCreateRepository
	config     config.ShortCodes
}

func NewShortCodeCreate(
	repository ShortCodeCreateRepository,
	config config.ShortCodes,
) *ShortCodeCreate {
	return &ShortCodeCreate{
		repository: repository,
		config:     config,
	}
}

func (service *ShortCodeCreate) Exec(
	ctx context.Context, request *ShortCodeCreateRequest,
) (*ShortCode, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ShortCodeCreate")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	// Generate a new random code.
	plainCode, err := lib.NewRandomURLString(service.config.Size)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("generate short code: %w", err))
	}

	// Encrypt the short code in the database.
	encrypted, err := lib.GenerateScrypt(plainCode, lib.ScryptParamsDefault)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("encrypt short code: %w", err))
	}

	// Serialize the data associated with the short code before storing it.
	var serializedData []byte
	if request.Data != nil {
		serializedData, err = json.Marshal(request.Data)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("serialize data: %w", err))
	}

	now := time.Now()
	expiry := now.Add(request.TTL)

	entity, err := service.repository.Exec(ctx, &dao.ShortCodeInsertRequest{
		ID:        uuid.New(),
		Code:      encrypted,
		Usage:     request.Usage,
		Target:    request.Target,
		Data:      serializedData,
		Now:       now,
		ExpiresAt: expiry,
		Override:  request.Override,
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	span.SetAttributes(attribute.String("shortCode.id", entity.ID.String()))

	return otel.ReportSuccess(span, &ShortCode{
		ID:        entity.ID,
		Usage:     entity.Usage,
		Target:    entity.Target,
		Data:      entity.Data,
		CreatedAt: entity.CreatedAt,
		ExpiresAt: entity.ExpiresAt,
		PlainCode: plainCode,
	}), nil
}
