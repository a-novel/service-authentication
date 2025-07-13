package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/models"
)

type UpdateEmailSource interface {
	UpdateCredentialsEmail(
		ctx context.Context, userID uuid.UUID, data dao.UpdateCredentialsEmailData,
	) (*dao.CredentialsEntity, error)
	ConsumeShortCode(ctx context.Context, request ConsumeShortCodeRequest) (*models.ShortCode, error)
}

func NewUpdateEmailSource(
	updateCredentialsEmail *dao.UpdateCredentialsEmailRepository,
	consumeShortCode *ConsumeShortCodeService,
) UpdateEmailSource {
	return &struct {
		*dao.UpdateCredentialsEmailRepository
		*ConsumeShortCodeService
	}{
		UpdateCredentialsEmailRepository: updateCredentialsEmail,
		ConsumeShortCodeService:          consumeShortCode,
	}
}

type UpdateEmailRequest struct {
	UserID    uuid.UUID
	ShortCode string
}

type UpdateEmailResponse struct {
	NewEmail string
}

type UpdateEmailService struct {
	source UpdateEmailSource
}

func NewUpdateEmailService(source UpdateEmailSource) *UpdateEmailService {
	return &UpdateEmailService{source: source}
}

func (service *UpdateEmailService) UpdateEmail(
	ctx context.Context, request UpdateEmailRequest,
) (*UpdateEmailResponse, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.UpdateEmail")
	defer span.End()

	span.SetAttributes(attribute.String("request.userID", request.UserID.String()))

	var (
		err         error
		credentials *dao.CredentialsEntity
	)

	err = postgres.RunInTx(ctx, nil, func(ctx context.Context, tx bun.IDB) error {
		// Verify short code.
		shortCode, err := service.source.ConsumeShortCode(ctx, ConsumeShortCodeRequest{
			Usage:  models.ShortCodeUsageValidateMail,
			Target: request.UserID.String(),
			Code:   request.ShortCode,
		})
		if err != nil {
			return fmt.Errorf("consume short code: %w", err)
		}

		var newEmail string

		err = json.Unmarshal(shortCode.Data, &newEmail)
		if err != nil {
			return fmt.Errorf("unmarshal short code data: %w", err)
		}

		span.SetAttributes(attribute.String("shortCode.newEmail", newEmail))

		// Update email.
		credentials, err = service.source.UpdateCredentialsEmail(ctx, request.UserID, dao.UpdateCredentialsEmailData{
			Email: newEmail,
			Now:   time.Now(),
		})
		if err != nil {
			return fmt.Errorf("update email: %w", err)
		}

		span.SetAttributes(attribute.String("dao.credentials.email", credentials.Email))

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return otel.ReportSuccess(span, &UpdateEmailResponse{
		NewEmail: credentials.Email,
	}), nil
}
