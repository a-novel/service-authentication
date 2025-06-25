package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/getsentry/sentry-go"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/models"
)

var (
	ErrMissingShortCodeAndCurrentPassword = errors.New("missing short code and current password")

	ErrUpdatePasswordService = errors.New("UpdatePasswordService.UpdatePassword")
)

func NewErrUpdatePasswordService(err error) error {
	return errors.Join(err, ErrUpdatePasswordService)
}

type UpdatePasswordSource interface {
	UpdateCredentialsPassword(
		ctx context.Context, userID uuid.UUID, data dao.UpdateCredentialsPasswordData,
	) (*dao.CredentialsEntity, error)
	SelectCredentials(ctx context.Context, id uuid.UUID) (*dao.CredentialsEntity, error)
	ConsumeShortCode(ctx context.Context, request ConsumeShortCodeRequest) (*models.ShortCode, error)
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

func (service *UpdatePasswordService) UpdatePassword(ctx context.Context, request UpdatePasswordRequest) error {
	span := sentry.StartSpan(ctx, "UpdatePasswordService.UpdatePassword")
	defer span.Finish()

	span.SetData("request.userID", request.UserID.String())

	// Encrypt the new password.
	encryptedPassword, err := lib.GenerateScrypt(request.Password, lib.ScryptParamsDefault)
	if err != nil {
		span.SetData("password.encrypt.error", err.Error())

		return NewErrUpdatePasswordService(fmt.Errorf("encrypt password: %w", err))
	}

	// Password update can fail after the short code is consumed. To prevent this, we wrap the operation in a single
	// transaction.
	ctxTx, commit, err := lib.PostgresContextTx(span.Context(), &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  false,
	})
	if err != nil {
		span.SetData("postgres.transaction.error", err.Error())

		return NewErrUpdatePasswordService(fmt.Errorf("create transaction: %w", err))
	}

	defer func() { _ = commit(false) }()

	switch {
	// If using a password reset, user does not have access to its original password. It should have issued a short
	// code to reset its password.
	case request.ShortCode != "":
		// Consume short code.
		_, err = service.source.ConsumeShortCode(ctxTx, ConsumeShortCodeRequest{
			Usage:  models.ShortCodeUsageResetPassword,
			Target: request.UserID.String(),
			Code:   request.ShortCode,
		})
		if err != nil {
			span.SetData("service.consumeShortCode.error", err.Error())

			return NewErrUpdatePasswordService(fmt.Errorf("consume short code: %w", err))
		}
	case request.CurrentPassword != "":
		credentials, err := service.source.SelectCredentials(ctxTx, request.UserID)
		if err != nil {
			span.SetData("dao.selectCredentials.error", err.Error())

			return NewErrUpdatePasswordService(fmt.Errorf("select credentials: %w", err))
		}

		// Check if the current password is correct.
		if err = lib.CompareScrypt(request.CurrentPassword, credentials.Password); err != nil {
			span.SetData("scrypt.compare.error", err.Error())

			return NewErrUpdatePasswordService(fmt.Errorf("compare current password: %w", err))
		}
	default:
		return ErrMissingShortCodeAndCurrentPassword
	}

	// Update password.
	_, err = service.source.UpdateCredentialsPassword(ctxTx, request.UserID, dao.UpdateCredentialsPasswordData{
		Password: encryptedPassword,
		Now:      time.Now(),
	})
	if err != nil {
		span.SetData("dao.updateCredentialsPassword.error", err.Error())

		return NewErrUpdatePasswordService(fmt.Errorf("update password: %w", err))
	}

	// Commit transaction.
	if err = commit(true); err != nil {
		span.SetData("postgres.commit.error", err.Error())

		return NewErrUpdatePasswordService(fmt.Errorf("commit transaction: %w", err))
	}

	return nil
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

func NewUpdatePasswordService(source UpdatePasswordSource) *UpdatePasswordService {
	return &UpdatePasswordService{source: source}
}
