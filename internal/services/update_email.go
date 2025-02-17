package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel-kit/context"
	pgctx "github.com/a-novel-kit/context/pgbun"

	"github.com/a-novel/authentication/internal/dao"
	"github.com/a-novel/authentication/models"
)

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
	// Email update can fail after the short code is consumed. To prevent this, we wrap the operation in a single
	// transaction.
	ctxTx, commit, err := pgctx.NewContextTX(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("(UpdateEmailService.UpdateEmail) create transaction: %w", err)
	}

	defer func() { _ = commit(false) }()

	// Verify short code.
	shortCode, err := service.source.ConsumeShortCode(ctxTx, ConsumeShortCodeRequest{
		Usage:  models.ShortCodeUsageValidateMail,
		Target: request.UserID.String(),
		Code:   request.ShortCode,
	})
	if err != nil {
		return nil, fmt.Errorf("(UpdateEmailService.UpdateEmail) consume short code: %w", err)
	}

	var newEmail string
	if err = json.Unmarshal(shortCode.Data, &newEmail); err != nil {
		return nil, fmt.Errorf("(UpdateEmailService.UpdateEmail) unmarshal short code data: %w", err)
	}

	// Update email.
	credentials, err := service.source.UpdateCredentialsEmail(ctxTx, request.UserID, dao.UpdateCredentialsEmailData{
		Email: newEmail,
		Now:   time.Now(),
	})
	if err != nil {
		return nil, fmt.Errorf("(UpdateEmailService.UpdateEmail) update email: %w", err)
	}

	// Commit transaction.
	if err = commit(true); err != nil {
		return nil, fmt.Errorf("(UpdateEmailService.UpdateEmail) commit transaction: %w", err)
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
