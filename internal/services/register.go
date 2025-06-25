package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/getsentry/sentry-go"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/models"
)

var ErrRegisterService = errors.New("RegisterService.Register")

func NewErrRegisterService(err error) error {
	return errors.Join(err, ErrRegisterService)
}

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
	span := sentry.StartSpan(ctx, "RegisterService.Register")
	defer span.Finish()

	span.SetData("email", request.Email)

	// Encrypt the password.
	encryptedPassword, err := lib.GenerateScrypt(request.Password, lib.ScryptParamsDefault)
	if err != nil {
		span.SetData("scrypt.error", err.Error())

		return "", NewErrRegisterService(fmt.Errorf("encrypt password: %w", err))
	}

	// Registration can fail after the short code is consumed. To prevent this, we wrap the operation in a single
	// transaction.
	ctxTx, commit, err := lib.PostgresContextTx(span.Context(), &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  false,
	})
	if err != nil {
		span.SetData("postgres.transaction.error", err.Error())

		return "", NewErrRegisterService(fmt.Errorf("create transaction: %w", err))
	}

	defer func() { _ = commit(false) }()

	// Verify short code.
	_, err = service.source.ConsumeShortCode(ctxTx, ConsumeShortCodeRequest{
		Usage:  models.ShortCodeUsageRequestRegister,
		Target: request.Email,
		Code:   request.ShortCode,
	})
	if err != nil {
		span.SetData("consumeShortCode.error", err.Error())

		return "", NewErrRegisterService(fmt.Errorf("consume short code: %w", err))
	}

	// Insert credentials.
	credentials, err := service.source.InsertCredentials(ctxTx, dao.InsertCredentialsData{
		ID:       uuid.New(),
		Email:    request.Email,
		Password: encryptedPassword,
		Now:      time.Now(),
	})
	if err != nil {
		span.SetData("dao.insertCredentials.error", err.Error())

		return "", NewErrRegisterService(fmt.Errorf("insert credentials: %w", err))
	}

	span.SetData("credentials.id", credentials.ID)

	// Commit transaction.
	if err = commit(true); err != nil {
		span.SetData("postgres.commit.error", err.Error())

		return "", NewErrRegisterService(fmt.Errorf("commit transaction: %w", err))
	}

	// Generate a new authentication token.
	accessToken, err := service.source.IssueToken(span.Context(), IssueTokenRequest{
		UserID: &credentials.ID,
		Roles: []models.Role{
			lo.Switch[models.CredentialsRole, models.Role](credentials.Role).
				Case(models.CredentialsRoleAdmin, models.RoleAdmin).
				Case(models.CredentialsRoleSuperAdmin, models.RoleSuperAdmin).
				Default(models.RoleUser),
		},
	})
	if err != nil {
		span.SetData("issueToken.error", err.Error())

		return "", NewErrRegisterService(fmt.Errorf("issue accessToken: %w", err))
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
