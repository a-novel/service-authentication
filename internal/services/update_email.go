package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/models"
)

var ErrUpdateEmailService = errors.New("UpdateEmailService.UpdateEmail")

func NewErrUpdateEmailService(err error) error {
	return errors.Join(err, ErrUpdateEmailService)
}

type UpdateEmailSource interface {
	UpdateCredentialsEmail(
		ctx context.Context, userID uuid.UUID, data dao.UpdateCredentialsEmailData,
	) (*dao.CredentialsEntity, error)
	ConsumeShortCode(ctx context.Context, request ConsumeShortCodeRequest) (*models.ShortCode, error)
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

func (service *UpdateEmailService) UpdateEmail(
	ctx context.Context, request UpdateEmailRequest,
) (*UpdateEmailResponse, error) {
	span := sentry.StartSpan(ctx, "UpdateEmailService.UpdateEmail")
	defer span.Finish()

	span.SetData("request.userID", request.UserID.String())

	// Email update can fail after the short code is consumed. To prevent this, we wrap the operation in a single
	// transaction.
	ctxTx, commit, err := lib.PostgresContextTx(span.Context(), &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  false,
	})
	if err != nil {
		span.SetData("postgres.transaction.error", err.Error())

		return nil, NewErrUpdateEmailService(fmt.Errorf("create transaction: %w", err))
	}

	defer func() { _ = commit(false) }()

	// Verify short code.
	shortCode, err := service.source.ConsumeShortCode(ctxTx, ConsumeShortCodeRequest{
		Usage:  models.ShortCodeUsageValidateMail,
		Target: request.UserID.String(),
		Code:   request.ShortCode,
	})
	if err != nil {
		span.SetData("service.consumeShortCode.error", err.Error())

		return nil, NewErrUpdateEmailService(fmt.Errorf("consume short code: %w", err))
	}

	var newEmail string
	if err = json.Unmarshal(shortCode.Data, &newEmail); err != nil {
		span.SetData("json.unmarshal.error", err.Error())

		return nil, NewErrUpdateEmailService(fmt.Errorf("unmarshal short code data: %w", err))
	}

	span.SetData("shortCode.newEmail", newEmail)

	// Update email.
	credentials, err := service.source.UpdateCredentialsEmail(ctxTx, request.UserID, dao.UpdateCredentialsEmailData{
		Email: newEmail,
		Now:   time.Now(),
	})
	if err != nil {
		span.SetData("dao.updateCredentialsEmail.error", err.Error())

		return nil, NewErrUpdateEmailService(fmt.Errorf("update email: %w", err))
	}

	span.SetData("dao.credentials.email", credentials.Email)

	// Commit transaction.
	if err = commit(true); err != nil {
		span.SetData("postgres.commit.error", err.Error())

		return nil, NewErrUpdateEmailService(fmt.Errorf("commit transaction: %w", err))
	}

	return &UpdateEmailResponse{
		NewEmail: credentials.Email,
	}, nil
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

func NewUpdateEmailService(source UpdateEmailSource) *UpdateEmailService {
	return &UpdateEmailService{source: source}
}
