package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/dao"
)

// CredentialsGetSelfDao loads the caller's account by ID.
type CredentialsGetSelfDao interface {
	Exec(ctx context.Context, request *dao.CredentialsSelectRequest) (*dao.Credentials, error)
}

// CredentialsGetSelfDaoShortCodeSelect loads the caller's active email-change code.
type CredentialsGetSelfDaoShortCodeSelect interface {
	Exec(ctx context.Context, request *dao.ShortCodeSelectRequest) (*dao.ShortCode, error)
}

// CredentialsGetSelfRequest names the authenticated user whose account to load.
type CredentialsGetSelfRequest struct {
	ID uuid.UUID
}

// CredentialsGetSelf retrieves the caller's own account, with the email change it awaits.
type CredentialsGetSelf struct {
	dao                CredentialsGetSelfDao
	daoShortCodeSelect CredentialsGetSelfDaoShortCodeSelect
}

func NewCredentialsGetSelf(
	dao CredentialsGetSelfDao, daoShortCodeSelect CredentialsGetSelfDaoShortCodeSelect,
) *CredentialsGetSelf {
	return &CredentialsGetSelf{dao: dao, daoShortCodeSelect: daoShortCodeSelect}
}

// Exec returns the account with a nil [CredentialsSelf.PendingEmail] when no change awaits
// confirmation. It returns [ErrCredentialsGetNotFound] when the account no longer exists.
func (service *CredentialsGetSelf) Exec(
	ctx context.Context, request *CredentialsGetSelfRequest,
) (*CredentialsSelf, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.CredentialsGetSelf")
	defer span.End()

	span.SetAttributes(attribute.String("user.id", request.ID.String()))

	entity, err := service.dao.Exec(ctx, &dao.CredentialsSelectRequest{ID: request.ID})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select credentials: %w", err))
	}

	credentials := &CredentialsSelf{Credentials: *newCredentials(entity)}

	shortCodeRequest := &dao.ShortCodeSelectRequest{
		Usage:  ShortCodeUsageValidateEmail,
		Target: request.ID.String(),
	}

	shortCode, err := service.daoShortCodeSelect.Exec(ctx, shortCodeRequest)
	if errors.Is(err, dao.ErrShortCodeSelectNotFound) {
		return credentials, nil
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select email update short code: %w", err))
	}

	credentials.PendingEmail = &PendingEmail{ExpiresAt: shortCode.ExpiresAt}

	err = json.Unmarshal(shortCode.Data, &credentials.PendingEmail.Email)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("unmarshal short code data: %w", err))
	}

	return credentials, nil
}
