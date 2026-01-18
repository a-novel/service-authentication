package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/lib"
)

type CredentialsUpdatePasswordRepository interface {
	Exec(
		ctx context.Context, request *dao.CredentialsUpdatePasswordRequest,
	) (*dao.Credentials, error)
}

type CredentialsUpdatePasswordRepositoryCredentialsSelect interface {
	Exec(ctx context.Context, request *dao.CredentialsSelectRequest) (*dao.Credentials, error)
}

type CredentialsUpdatePasswordServiceShortCodeConsume interface {
	Exec(ctx context.Context, request *ShortCodeConsumeRequest) (*ShortCode, error)
}

type CredentialsUpdatePasswordRequest struct {
	Password        string `validate:"required,min=4,max=1024"`
	CurrentPassword string `validate:"required_without=ShortCode,max=1024"`
	ShortCode       string `validate:"required_without=CurrentPassword,max=1024"`
	UserID          uuid.UUID
}

type CredentialsUpdatePassword struct {
	repository                  CredentialsUpdatePasswordRepository
	repositoryCredentialsSelect CredentialsUpdatePasswordRepositoryCredentialsSelect
	serviceShortCodeConsume     CredentialsUpdatePasswordServiceShortCodeConsume
}

func NewCredentialsUpdatePassword(
	repository CredentialsUpdatePasswordRepository,
	repositoryCredentialsSelect CredentialsUpdatePasswordRepositoryCredentialsSelect,
	serviceShortCodeConsume CredentialsUpdatePasswordServiceShortCodeConsume,
) *CredentialsUpdatePassword {
	return &CredentialsUpdatePassword{
		repository:                  repository,
		repositoryCredentialsSelect: repositoryCredentialsSelect,
		serviceShortCodeConsume:     serviceShortCodeConsume,
	}
}

func (service *CredentialsUpdatePassword) Exec(
	ctx context.Context, request *CredentialsUpdatePasswordRequest,
) (*Credentials, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.CredentialsUpdatePassword")
	defer span.End()

	span.SetAttributes(attribute.String("request.userID", request.UserID.String()))

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	// Encrypt the new password.
	encryptedPassword, err := lib.GenerateArgon2(request.Password, lib.Argon2ParamsDefault)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("encrypt password: %w", err))
	}

	var credentials *dao.Credentials

	// Password update supports two authentication paths, running within a transaction
	// to ensure atomicity of verification and update.
	err = postgres.RunInTx(ctx, nil, func(ctx context.Context, tx bun.IDB) error {
		switch {
		// Path 1: Password Reset Flow
		// Used when the user forgot their password and requested a reset via email.
		// The short code proves the user owns the email address associated with this account.
		// This path does NOT require knowing the current password.
		case request.ShortCode != "":
			_, err = service.serviceShortCodeConsume.Exec(ctx, &ShortCodeConsumeRequest{
				Usage:  ShortCodeUsageResetPassword,
				Target: request.UserID.String(),
				Code:   request.ShortCode,
			})
			if err != nil {
				return fmt.Errorf("consume short code: %w", err)
			}

		// Path 2: Authenticated Password Change
		// Used when a logged-in user wants to change their password.
		// Requires the current password as proof of identity, preventing
		// attackers with session access from locking out the legitimate user.
		case request.CurrentPassword != "":
			credentials, err = service.repositoryCredentialsSelect.Exec(
				ctx,
				&dao.CredentialsSelectRequest{ID: request.UserID},
			)
			if err != nil {
				return fmt.Errorf("select credentials: %w", err)
			}

			err = lib.CompareArgon2(request.CurrentPassword, credentials.Password)
			if err != nil {
				return fmt.Errorf("compare current password: %w", err)
			}
		}

		// Update the password with the new Argon2 hash.
		credentials, err = service.repository.Exec(ctx, &dao.CredentialsUpdatePasswordRequest{
			ID:       request.UserID,
			Password: encryptedPassword,
			Now:      time.Now(),
		})
		if err != nil {
			return fmt.Errorf("update password: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	otel.ReportSuccessNoContent(span)

	return &Credentials{
		ID:        credentials.ID,
		Email:     credentials.Email,
		Role:      credentials.Role,
		CreatedAt: credentials.CreatedAt,
		UpdatedAt: credentials.UpdatedAt,
	}, nil
}
