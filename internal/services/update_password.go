package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/models"
)

var ErrMissingShortCodeAndCurrentPassword = errors.New("missing short code and current password")

type UpdatePasswordSource interface {
	UpdateCredentialsPassword(
		ctx context.Context, userID uuid.UUID, data dao.UpdateCredentialsPasswordData,
	) (*dao.CredentialsEntity, error)
	SelectCredentials(ctx context.Context, id uuid.UUID) (*dao.CredentialsEntity, error)
	ConsumeShortCode(ctx context.Context, request ConsumeShortCodeRequest) (*models.ShortCode, error)
}

func NewUpdatePasswordSource(
	selectCredentialsDAO *dao.SelectCredentialsRepository,
	updateCredentialsDAO *dao.UpdateCredentialsPasswordRepository,
	consumeShortCode *ConsumeShortCodeService,
) UpdatePasswordSource {
	return &struct {
		*dao.SelectCredentialsRepository
		*dao.UpdateCredentialsPasswordRepository
		*ConsumeShortCodeService
	}{
		SelectCredentialsRepository:         selectCredentialsDAO,
		UpdateCredentialsPasswordRepository: updateCredentialsDAO,
		ConsumeShortCodeService:             consumeShortCode,
	}
}

type UpdatePasswordRequest struct {
	Password        string
	CurrentPassword string
	ShortCode       string
	UserID          uuid.UUID
}

type UpdatePasswordService struct {
	source UpdatePasswordSource
}

func NewUpdatePasswordService(source UpdatePasswordSource) *UpdatePasswordService {
	return &UpdatePasswordService{source: source}
}

func (service *UpdatePasswordService) UpdatePassword(ctx context.Context, request UpdatePasswordRequest) error {
	ctx, span := otel.Tracer().Start(ctx, "service.UpdatePassword")
	defer span.End()

	span.SetAttributes(attribute.String("request.userID", request.UserID.String()))

	// Encrypt the new password.
	encryptedPassword, err := lib.GenerateScrypt(request.Password, lib.ScryptParamsDefault)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("encrypt password: %w", err))
	}

	err = postgres.RunInTx(ctx, nil, func(ctx context.Context, tx bun.IDB) error {
		switch {
		// If using a password reset, user does not have access to its original password. It should have issued a short
		// code to reset its password.
		case request.ShortCode != "":
			// Consume short code.
			_, err = service.source.ConsumeShortCode(ctx, ConsumeShortCodeRequest{
				Usage:  models.ShortCodeUsageResetPassword,
				Target: request.UserID.String(),
				Code:   request.ShortCode,
			})
			if err != nil {
				return fmt.Errorf("consume short code: %w", err)
			}
		case request.CurrentPassword != "":
			credentials, err := service.source.SelectCredentials(ctx, request.UserID)
			if err != nil {
				return fmt.Errorf("select credentials: %w", err)
			}

			// Check if the current password is correct.
			err = lib.CompareScrypt(request.CurrentPassword, credentials.Password)
			if err != nil {
				return fmt.Errorf("compare current password: %w", err)
			}
		default:
			return ErrMissingShortCodeAndCurrentPassword
		}

		// Update password.
		_, err = service.source.UpdateCredentialsPassword(ctx, request.UserID, dao.UpdateCredentialsPasswordData{
			Password: encryptedPassword,
			Now:      time.Now(),
		})
		if err != nil {
			return fmt.Errorf("update password: %w", err)
		}

		return nil
	})
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	otel.ReportSuccessNoContent(span)

	return nil
}
