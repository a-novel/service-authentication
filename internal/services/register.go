package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/a-novel-kit/jwt/jwa"

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
	IssueRefreshToken(ctx context.Context, request IssueRefreshTokenRequest) (string, *jwa.Claims, error)
	ConsumeShortCode(ctx context.Context, request ConsumeShortCodeRequest) (*models.ShortCode, error)
}

func NewRegisterSource(
	insertCredentialsDAO *dao.InsertCredentialsRepository,
	issueTokenService *IssueTokenService,
	issueRefreshTokenService *IssueRefreshTokenService,
	consumeShortCodeService *ConsumeShortCodeService,
) RegisterSource {
	return &struct {
		*dao.InsertCredentialsRepository
		*IssueTokenService
		*ConsumeShortCodeService
		*IssueRefreshTokenService
	}{
		InsertCredentialsRepository: insertCredentialsDAO,
		IssueTokenService:           issueTokenService,
		IssueRefreshTokenService:    issueRefreshTokenService,
		ConsumeShortCodeService:     consumeShortCodeService,
	}
}

type RegisterRequest struct {
	Email     string
	Password  string
	ShortCode string
}

type RegisterService struct {
	source RegisterSource
}

func NewRegisterService(source RegisterSource) *RegisterService {
	return &RegisterService{source: source}
}

func (service *RegisterService) Register(ctx context.Context, request RegisterRequest) (*models.Token, error) {
	span := sentry.StartSpan(ctx, "RegisterService.Register")
	defer span.Finish()

	span.SetData("email", request.Email)

	// Encrypt the password.
	encryptedPassword, err := lib.GenerateScrypt(request.Password, lib.ScryptParamsDefault)
	if err != nil {
		span.SetData("scrypt.error", err.Error())

		return nil, NewErrRegisterService(fmt.Errorf("encrypt password: %w", err))
	}

	// Registration can fail after the short code is consumed. To prevent this, we wrap the operation in a single
	// transaction.
	ctxTx, commit, err := lib.PostgresContextTx(span.Context(), &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  false,
	})
	if err != nil {
		span.SetData("postgres.transaction.error", err.Error())

		return nil, NewErrRegisterService(fmt.Errorf("create transaction: %w", err))
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

		return nil, NewErrRegisterService(fmt.Errorf("consume short code: %w", err))
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

		return nil, NewErrRegisterService(fmt.Errorf("insert credentials: %w", err))
	}

	span.SetData("credentials.id", credentials.ID)

	// Commit transaction.
	err = commit(true)
	if err != nil {
		span.SetData("postgres.commit.error", err.Error())

		return nil, NewErrRegisterService(fmt.Errorf("commit transaction: %w", err))
	}

	refreshToken, refreshTokenClaims, err := service.source.IssueRefreshToken(span.Context(), IssueRefreshTokenRequest{
		Claims: &models.AccessTokenClaims{
			UserID: &credentials.ID,
			Roles: []models.Role{
				lo.Switch[models.CredentialsRole, models.Role](credentials.Role).
					Case(models.CredentialsRoleAdmin, models.RoleAdmin).
					Case(models.CredentialsRoleSuperAdmin, models.RoleSuperAdmin).
					Default(models.RoleUser),
			},
		},
	})
	if err != nil {
		span.SetData("issueRefreshToken.error", err.Error())

		return nil, NewErrLoginService(fmt.Errorf("issue refresh token: %w", err))
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
		RefreshTokenID: &refreshTokenClaims.Jti,
	})
	if err != nil {
		span.SetData("issueToken.error", err.Error())

		return nil, NewErrRegisterService(fmt.Errorf("issue accessToken: %w", err))
	}

	return &models.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
