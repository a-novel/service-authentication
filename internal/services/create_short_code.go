package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/models"
	"github.com/a-novel/service-authentication/models/config"
)

// CreateShortCodeSource is the source used to perform the CreateShortCodeService.CreateShortCode action.
type CreateShortCodeSource interface {
	InsertShortCode(ctx context.Context, data dao.InsertShortCodeData) (*dao.ShortCodeEntity, error)
}

// CreateShortCodeRequest is the input used to perform the CreateShortCodeService.CreateShortCode action.
type CreateShortCodeRequest struct {
	// Information about the resource the short code grants access to.
	Usage models.ShortCodeUsage
	// The target that is allowed to access the resource. Only this target can retrieve the short code.
	Target string
	// Data used for the targeted resource. It can contain any information required to perform a specific action.
	Data any
	// TTL (Time To Live) of the short code. Past this delay, the short code expires, and can no longer be used.
	TTL time.Duration

	// Whe true, automatically expires any existing short code with the same target and usage.
	// Otherwise, the presence of a duplica will trigger an error.
	Override bool
}

// CreateShortCodeService is the service used to perform the CreateShortCodeService.CreateShortCode action.
//
// You may create one using the NewCreateShortCodeService function.
type CreateShortCodeService struct {
	source CreateShortCodeSource
	config config.ShortCodes
}

func NewCreateShortCodeService(source CreateShortCodeSource, config config.ShortCodes) *CreateShortCodeService {
	return &CreateShortCodeService{source: source, config: config}
}

// CreateShortCode creates a new short code in the database.
//
// This service automatically generates a random short code, encrypts it, and stores it in the database. The clear
// value of the short code is returned, and MUST be sent to the target using a private channel (usually mail or phone).
//
// The clear value is URL-safe. If any encoding is performed on it, then it MUST be decoded before being sent back for
// consumption.
func (service *CreateShortCodeService) CreateShortCode(
	ctx context.Context, request CreateShortCodeRequest,
) (*models.ShortCode, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.CreateShortCode")
	defer span.End()

	span.SetAttributes(
		attribute.String("request.target", request.Target),
		attribute.String("request.usage", request.Usage.String()),
		attribute.Int64("request.ttl", request.TTL.Milliseconds()),
		attribute.Bool("request.override", request.Override),
	)

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

	entity, err := service.source.InsertShortCode(ctx, dao.InsertShortCodeData{
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
		return nil, otel.ReportError(span, fmt.Errorf("insert short code: %w", err))
	}

	span.SetAttributes(attribute.String("shortCode.id", entity.ID.String()))

	return otel.ReportSuccess(span, &models.ShortCode{
		ID:        entity.ID,
		Usage:     entity.Usage,
		Target:    entity.Target,
		Data:      entity.Data,
		CreatedAt: entity.CreatedAt,
		ExpiresAt: entity.ExpiresAt,
		PlainCode: plainCode,
	}), nil
}
