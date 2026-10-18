package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/transaction"

	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/lib"
)

type CredentialsUpdatePasswordDao interface {
	Exec(
		ctx context.Context, request *dao.CredentialsUpdatePasswordRequest,
	) (*dao.Credentials, error)
}

type CredentialsUpdatePasswordDaoCredentialsSelect interface {
	Exec(ctx context.Context, request *dao.CredentialsSelectRequest) (*dao.Credentials, error)
}

type CredentialsUpdatePasswordServiceShortCodeVerify interface {
	Exec(ctx context.Context, request *ShortCodeVerifyRequest) (*ShortCode, error)
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

// CredentialsUpdatePassword changes an account's password through one of two
// authenticated paths: a reset short code proving the caller owns the account's
// email, or the current password proving an active session belongs to the owner.
type CredentialsUpdatePassword struct {
	dao                     CredentialsUpdatePasswordDao
	daoCredentialsSelect    CredentialsUpdatePasswordDaoCredentialsSelect
	serviceShortCodeVerify  CredentialsUpdatePasswordServiceShortCodeVerify
	serviceShortCodeConsume CredentialsUpdatePasswordServiceShortCodeConsume
	transactor              transaction.Transactor
}

func NewCredentialsUpdatePassword(
	dao CredentialsUpdatePasswordDao,
	daoCredentialsSelect CredentialsUpdatePasswordDaoCredentialsSelect,
	serviceShortCodeVerify CredentialsUpdatePasswordServiceShortCodeVerify,
	serviceShortCodeConsume CredentialsUpdatePasswordServiceShortCodeConsume,
	transactor transaction.Transactor,
) *CredentialsUpdatePassword {
	return &CredentialsUpdatePassword{
		dao:                     dao,
		daoCredentialsSelect:    daoCredentialsSelect,
		serviceShortCodeVerify:  serviceShortCodeVerify,
		serviceShortCodeConsume: serviceShortCodeConsume,
		transactor:              transactor,
	}
}

func (service *CredentialsUpdatePassword) Exec(
	ctx context.Context, request *CredentialsUpdatePasswordRequest,
) (*Credentials, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.CredentialsUpdatePassword")
	defer span.End()

	span.SetAttributes(attribute.String("user.id", request.UserID.String()))

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	var (
		credentials *dao.Credentials
		shortCode   *ShortCode
	)

	// Argon2 runs before the transaction, which holds a pooled connection: the proof first, so a
	// failed check costs a single run, then the new password.
	switch {
	// Reset path: the short code proves the caller owns the account's email, so no
	// current password is required.
	case request.ShortCode != "":
		shortCode, err = service.serviceShortCodeVerify.Exec(ctx, &ShortCodeVerifyRequest{
			Usage:  ShortCodeUsageResetPassword,
			Target: request.UserID.String(),
			Code:   request.ShortCode,
		})
		if err != nil {
			return nil, otel.ReportError(span, fmt.Errorf("verify short code: %w", err))
		}

	// Change path: verifying the current password stops someone holding only a live
	// session from locking the owner out of their own account.
	case request.CurrentPassword != "":
		credentials, err = service.daoCredentialsSelect.Exec(ctx, &dao.CredentialsSelectRequest{ID: request.UserID})
		if err != nil {
			return nil, otel.ReportError(span, fmt.Errorf("select credentials: %w", err))
		}

		err = lib.CompareArgon2(request.CurrentPassword, credentials.Password)
		if err != nil {
			return nil, otel.ReportError(span, fmt.Errorf("compare current password: %w", err))
		}
	}

	encryptedPassword, err := lib.GenerateArgon2(request.Password, lib.Argon2ParamsDefault)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("encrypt password: %w", err))
	}

	// Retiring the code and writing the password share one transaction, so a failed
	// write leaves the code usable.
	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		if shortCode != nil {
			_, err = service.serviceShortCodeConsume.Exec(ctx, &ShortCodeConsumeRequest{ID: shortCode.ID})
			if err != nil {
				return fmt.Errorf("consume short code: %w", err)
			}
		}

		credentials, err = service.dao.Exec(ctx, &dao.CredentialsUpdatePasswordRequest{
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

	return &Credentials{
		ID:        credentials.ID,
		Email:     credentials.Email,
		Role:      credentials.Role,
		CreatedAt: credentials.CreatedAt,
		UpdatedAt: credentials.UpdatedAt,
	}, nil
}
