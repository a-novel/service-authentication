package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-authentication/v2/internal/dao"
)

type CredentialsUpdateEmailRepository interface {
	Exec(ctx context.Context, request *dao.CredentialsUpdateEmailRequest) (*dao.Credentials, error)
}
type CredentialsUpdateEmailServiceShortCodeConsume interface {
	Exec(ctx context.Context, request *ShortCodeConsumeRequest) (*ShortCode, error)
}

type CredentialsUpdateEmailRequest struct {
	UserID    uuid.UUID
	ShortCode string `validate:"required,max=512"`
}

type CredentialsUpdateEmail struct {
	repository              CredentialsUpdateEmailRepository
	serviceShortCodeConsume CredentialsUpdateEmailServiceShortCodeConsume
}

func NewCredentialsUpdateEmail(
	repository CredentialsUpdateEmailRepository,
	serviceShortCodeConsume CredentialsUpdateEmailServiceShortCodeConsume,
) *CredentialsUpdateEmail {
	return &CredentialsUpdateEmail{
		repository:              repository,
		serviceShortCodeConsume: serviceShortCodeConsume,
	}
}

func (service *CredentialsUpdateEmail) Exec(
	ctx context.Context, request *CredentialsUpdateEmailRequest,
) (*Credentials, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.CredentialsUpdateEmail")
	defer span.End()

	span.SetAttributes(attribute.String("request.userID", request.UserID.String()))

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	var credentials *dao.Credentials

	err = postgres.RunInTx(ctx, nil, func(ctx context.Context, tx bun.IDB) error {
		// Verify short code.
		shortCode, txErr := service.serviceShortCodeConsume.Exec(ctx, &ShortCodeConsumeRequest{
			Usage:  ShortCodeUsageValidateEmail,
			Target: request.UserID.String(),
			Code:   request.ShortCode,
		})
		if txErr != nil {
			return fmt.Errorf("consume short code: %w", txErr)
		}

		var newEmail string

		txErr = json.Unmarshal(shortCode.Data, &newEmail)
		if txErr != nil {
			return fmt.Errorf("unmarshal short code data: %w", txErr)
		}

		span.SetAttributes(attribute.String("shortCode.newEmail", newEmail))

		// Update email.
		credentials, txErr = service.repository.Exec(ctx, &dao.CredentialsUpdateEmailRequest{
			ID:    request.UserID,
			Email: newEmail,
			Now:   time.Now(),
		})
		if txErr != nil {
			return fmt.Errorf("update email: %w", txErr)
		}

		span.SetAttributes(attribute.String("dao.credentials.email", credentials.Email))

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return otel.ReportSuccess(span, &Credentials{
		ID:        credentials.ID,
		Email:     credentials.Email,
		Role:      credentials.Role,
		CreatedAt: credentials.CreatedAt,
		UpdatedAt: credentials.UpdatedAt,
	}), nil
}
