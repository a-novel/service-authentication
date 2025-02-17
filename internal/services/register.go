package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel-kit/context"
	pgctx "github.com/a-novel-kit/context/pgbun"

	"github.com/a-novel/authentication/internal/dao"
	"github.com/a-novel/authentication/internal/lib"
	"github.com/a-novel/authentication/models"
)

type RegisterSource interface {
	InsertCredentials(ctx context.Context, data dao.InsertCredentialsData) (*dao.CredentialsEntity, error)
	IssueToken(ctx context.Context, request IssueTokenRequest) (string, error)
	ConsumeShortCode(ctx context.Context, request ConsumeShortCodeRequest) (*models.ShortCode, error)
}

type RegisterRequest struct {
	Email     string
	Password  string
	ShortCode string
}

type RegisterService struct {
	source RegisterSource
}

func (service *RegisterService) Register(ctx context.Context, request RegisterRequest) (string, error) {
	// Encrypt the password.
	encryptedPassword, err := lib.GenerateScrypt(request.Password, lib.ScryptParamsDefault)
	if err != nil {
		return "", fmt.Errorf("(RegisterService.Register) encrypt password: %w", err)
	}

	// Registration can fail after the short code is consumed. To prevent this, we wrap the operation in a single
	// transaction.
	ctxTx, commit, err := pgctx.NewContextTX(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("(RegisterService.Register) create transaction: %w", err)
	}

	defer func() { _ = commit(false) }()

	// Verify short code.
	_, err = service.source.ConsumeShortCode(ctxTx, ConsumeShortCodeRequest{
		Usage:  models.ShortCodeUsageRequestRegister,
		Target: request.Email,
		Code:   request.ShortCode,
	})
	if err != nil {
		return "", fmt.Errorf("(RegisterService.Register) consume short code: %w", err)
	}

	// Insert credentials.
	credentials, err := service.source.InsertCredentials(ctxTx, dao.InsertCredentialsData{
		ID:       uuid.New(),
		Email:    request.Email,
		Password: encryptedPassword,
		Now:      time.Now(),
	})
	if err != nil {
		return "", fmt.Errorf("(RegisterService.Register) insert credentials: %w", err)
	}

	// Commit transaction.
	if err = commit(true); err != nil {
		return "", fmt.Errorf("(RegisterService.Register) commit transaction: %w", err)
	}

	// Generate a new authentication token.
	accessToken, err := service.source.IssueToken(ctx, IssueTokenRequest{
		UserID: &credentials.ID,
		Roles:  []models.Role{models.RoleUser},
	})
	if err != nil {
		return "", fmt.Errorf("(RegisterService.Register) issue accessToken: %w", err)
	}

	return accessToken, nil
}

func NewRegisterSource(
	insertCredentialsDAO *dao.InsertCredentialsRepository,
	issueTokenService *IssueTokenService,
	consumeShortCodeService *ConsumeShortCodeService,
) RegisterSource {
	return &struct {
		*dao.InsertCredentialsRepository
		*IssueTokenService
		*ConsumeShortCodeService
	}{
		InsertCredentialsRepository: insertCredentialsDAO,
		IssueTokenService:           issueTokenService,
		ConsumeShortCodeService:     consumeShortCodeService,
	}
}

func NewRegisterService(source RegisterSource) *RegisterService {
	return &RegisterService{source: source}
}
