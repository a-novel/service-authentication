package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"
	jkModels "github.com/a-novel/service-json-keys/models"
	jkPkg "github.com/a-novel/service-json-keys/pkg"

	"github.com/a-novel-kit/jwt"
	"github.com/a-novel-kit/jwt/jws"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/models"
)

type RegisterSource interface {
	InsertCredentials(ctx context.Context, data dao.InsertCredentialsData) (*dao.CredentialsEntity, error)
	SignClaims(ctx context.Context, usage jkModels.KeyUsage, claims any) (string, error)
	ConsumeShortCode(ctx context.Context, request ConsumeShortCodeRequest) (*models.ShortCode, error)
}

func NewRegisterSource(
	insertCredentialsDAO *dao.InsertCredentialsRepository,
	issueTokenService *jkPkg.ClaimsSigner,
	consumeShortCodeService *ConsumeShortCodeService,
) RegisterSource {
	return &struct {
		*dao.InsertCredentialsRepository
		*ConsumeShortCodeService
		*jkPkg.ClaimsSigner
	}{
		InsertCredentialsRepository: insertCredentialsDAO,
		ClaimsSigner:                issueTokenService,
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
	ctx, span := otel.Tracer().Start(ctx, "service.Register")
	defer span.End()

	span.SetAttributes(attribute.String("email", request.Email))

	// Encrypt the password.
	encryptedPassword, err := lib.GenerateScrypt(request.Password, lib.ScryptParamsDefault)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("encrypt password: %w", err))
	}

	var credentials *dao.CredentialsEntity

	err = postgres.RunInTx(ctx, nil, func(ctx context.Context, tx bun.IDB) error {
		// Verify short code.
		_, err = service.source.ConsumeShortCode(ctx, ConsumeShortCodeRequest{
			Usage:  models.ShortCodeUsageRequestRegister,
			Target: request.Email,
			Code:   request.ShortCode,
		})
		if err != nil {
			return fmt.Errorf("consume short code: %w", err)
		}

		// Insert credentials.
		credentials, err = service.source.InsertCredentials(ctx, dao.InsertCredentialsData{
			ID:       uuid.New(),
			Email:    request.Email,
			Password: encryptedPassword,
			Now:      time.Now(),
		})
		if err != nil {
			return fmt.Errorf("insert credentials: %w", err)
		}

		span.SetAttributes(attribute.String("credentials.id", credentials.ID.String()))

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	refreshToken, err := service.source.SignClaims(
		ctx,
		jkModels.KeyUsageRefresh,
		models.RefreshTokenClaimsInput{
			UserID: credentials.ID,
		},
	)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("issue refresh token: %w", err))
	}

	refreshTokenRecipient := jwt.NewRecipient(jwt.RecipientConfig{
		Plugins: []jwt.RecipientPlugin{jws.NewInsecureVerifier()},
	})

	var refreshTokenClaims models.RefreshTokenClaims

	err = refreshTokenRecipient.Consume(ctx, refreshToken, &refreshTokenClaims)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("unmarshal refresh token claims: %w", err))
	}

	// Generate a new authentication token.
	accessToken, err := service.source.SignClaims(
		ctx,
		jkModels.KeyUsageAuth,
		models.AccessTokenClaims{
			UserID: &credentials.ID,
			Roles: []models.Role{
				lo.Switch[models.CredentialsRole, models.Role](credentials.Role).
					Case(models.CredentialsRoleAdmin, models.RoleAdmin).
					Case(models.CredentialsRoleSuperAdmin, models.RoleSuperAdmin).
					Default(models.RoleUser),
			},
			RefreshTokenID: &refreshTokenClaims.Jti,
		},
	)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("issue accessToken: %w", err))
	}

	return otel.ReportSuccess(span, &models.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}), nil
}
