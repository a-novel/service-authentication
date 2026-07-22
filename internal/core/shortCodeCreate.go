package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/lib"
)

// ShortCodeCreateDao persists a new short code and returns the stored row.
type ShortCodeCreateDao interface {
	Exec(ctx context.Context, request *dao.ShortCodeInsertRequest) (*dao.ShortCode, error)
}

// ShortCodeCreateRequest describes a code to issue: the flow it is for, the
// subject it binds to, any flow-specific data to carry, and how long it lives.
type ShortCodeCreateRequest struct {
	Usage  string `validate:"required,usage"`
	Target string `validate:"required,max=1024"`
	Data   any
	TTL    time.Duration `validate:"required"`

	// Override expires and replaces any existing code for the same target and usage.
	// When false, a duplicate fails the request.
	Override bool
}

// ShortCodeCreate issues a short code: it generates a random plaintext code,
// stores only its Argon2id hash, and returns the plaintext once so the caller can
// deliver it to the target. [ShortCodeConsume] redeems it.
type ShortCodeCreate struct {
	dao    ShortCodeCreateDao
	config config.ShortCodes
}

// NewShortCodeCreate wires the create service to its DAO and short-code configuration.
func NewShortCodeCreate(
	dao ShortCodeCreateDao,
	config config.ShortCodes,
) *ShortCodeCreate {
	return &ShortCodeCreate{
		dao:    dao,
		config: config,
	}
}

// Exec issues a new short code and returns it with the plaintext populated in
// [ShortCode.PlainCode]; only the hash is stored.
func (service *ShortCodeCreate) Exec(
	ctx context.Context, request *ShortCodeCreateRequest,
) (*ShortCode, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ShortCodeCreate")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	plainCode, err := lib.NewRandomURLString(service.config.Size)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("generate short code: %w", err))
	}

	// Store only the Argon2id hash; the plaintext code never reaches the database.
	encrypted, err := lib.GenerateArgon2(plainCode, lib.Argon2ParamsDefault)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("encrypt short code: %w", err))
	}

	var serializedData []byte
	if request.Data != nil {
		serializedData, err = json.Marshal(request.Data)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("serialize data: %w", err))
	}

	now := time.Now()
	expiry := now.Add(request.TTL)

	entity, err := service.dao.Exec(ctx, &dao.ShortCodeInsertRequest{
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
