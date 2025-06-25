package services

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/getsentry/sentry-go"
	"time"

	"github.com/go-faster/errors"
	"github.com/google/uuid"

	"github.com/a-novel/service-authentication/config"
	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/models"
)

var ErrCreateShortCodeService = errors.New("CreateShortCodeService.CreateShortCode")

func NewErrCreateShortCodeService(err error) error {
	return errors.Join(err, ErrCreateShortCodeService)
}

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
	span := sentry.StartSpan(ctx, "CreateShortCodeService.CreateShortCode")
	defer span.Finish()

	span.SetData("request.target", request.Target)
	span.SetData("request.usage", request.Usage)
	span.SetData("request.ttl", request.TTL)
	span.SetData("request.override", request.Override)

	// Generate a new random code.
	plainCode, err := lib.NewRandomURLString(config.ShortCodes.Size)
	if err != nil {
		span.SetData("generateRandomText.error", err.Error())

		return nil, NewErrCreateShortCodeService(fmt.Errorf("generate short code: %w", err))
	}

	// Encrypt the short code in the database.
	encrypted, err := lib.GenerateScrypt(plainCode, lib.ScryptParamsDefault)
	if err != nil {
		span.SetData("scrypt.error", err.Error())

		return nil, NewErrCreateShortCodeService(fmt.Errorf("encrypt short code: %w", err))
	}

	// Serialize the data associated with the short code before storing it.
	var serializedData []byte
	if request.Data != nil {
		serializedData, err = json.Marshal(request.Data)
	}

	if err != nil {
		span.SetData("json.serialize.error", err.Error())

		return nil, NewErrCreateShortCodeService(fmt.Errorf("serialize data: %w", err))
	}

	now := time.Now()
	expiry := now.Add(request.TTL)

	entity, err := service.source.InsertShortCode(span.Context(), dao.InsertShortCodeData{
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
		span.SetData("dao.insertShortCode.error", err.Error())

		return nil, NewErrCreateShortCodeService(fmt.Errorf("insert short code: %w", err))
	}

	span.SetData("shortCode.id", entity.ID)

	return &models.ShortCode{
		ID:        entity.ID,
		Usage:     entity.Usage,
		Target:    entity.Target,
		Data:      entity.Data,
		CreatedAt: entity.CreatedAt,
		ExpiresAt: entity.ExpiresAt,
		PlainCode: plainCode,
	}, nil
}

func NewCreateShortCodeService(source CreateShortCodeSource) *CreateShortCodeService {
	return &CreateShortCodeService{source: source}
}
