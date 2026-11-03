package core

import (
	"context"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/otel"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

// Credentials is a user account as the core layer exposes it: identity, email,
// current role, and audit timestamps. The stored password hash stays in the DAO
// layer and is never carried on this type.
type Credentials struct {
	ID        uuid.UUID
	Email     string
	Role      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CredentialsUpdate is an account after its password or email changed. The change ends every session
// of the account opened before it, and Token keeps the caller's own. Token is nil when no session of
// the account made the request, as for a reset.
type CredentialsUpdate struct {
	Credentials

	Token *Token
}

// CredentialsSelf is the caller's own account, as [CredentialsGetSelf] returns it.
type CredentialsSelf struct {
	Credentials

	// PendingEmail is the email change awaiting confirmation, nil when none does.
	PendingEmail *PendingEmail
}

// PendingEmail is an email change requested through [ShortCodeCreateEmailUpdate]. It lasts until
// the user confirms it, requests another change, or lets the confirmation code expire.
type PendingEmail struct {
	// Email is the new address, which takes effect once confirmed.
	Email string

	// ExpiresAt is when the confirmation code expires.
	ExpiresAt time.Time
}

// ValidateCredentialsRole is a go-playground/validator field-level validator that
// accepts a string only when it names a role defined in the default permissions
// configuration. It is registered under the "role" tag at package init.
func ValidateCredentialsRole(fl validator.FieldLevel) bool {
	_, ok := authconfig.PermissionsConfigDefault.Roles[fl.Field().String()]

	return ok
}

func newCredentials(entity *dao.Credentials) *Credentials {
	return &Credentials{
		ID:        entity.ID,
		Email:     entity.Email,
		Role:      entity.Role,
		CreatedAt: entity.CreatedAt,
		UpdatedAt: entity.UpdatedAt,
	}
}

// credentialsChangedSince reports whether the password or email of credentials changed after a
// token issued at issuedAt, a Unix timestamp in seconds. Such a token opens no session any more: a
// change ends every session opened before it. Both instants count whole seconds, so a token issued
// within the second of the change keeps working, and the person signs in again right after it.
func credentialsChangedSince(credentials *dao.Credentials, issuedAt int64) bool {
	return issuedAt < credentials.CredentialsUpdatedAt.Unix()
}

// keepSession returns the account after a password or email change, with the token pair that keeps
// caller's session past it when caller holds a session of the account. The pair joins that session
// and ends with it, so a change never extends a session. A failed signature leaves Token nil: the
// change stands, and the caller signs in again.
func keepSession(
	ctx context.Context, signer tokenPairSigner, credentials *dao.Credentials, caller *session.Claims,
) *CredentialsUpdate {
	ctx, span := otel.Tracer().Start(ctx, "core.keepSession")
	defer span.End()

	result := &CredentialsUpdate{Credentials: *newCredentials(credentials)}

	if caller == nil || caller.Session() == "" || lo.FromPtr(caller.UserID) != credentials.ID {
		return result
	}

	refreshClaims := session.RefreshClaimsForm{SessionID: caller.Session(), SessionExp: caller.SessionExp}

	// signTokenPair reports its failure on its own span, and the change stands without a pair.
	result.Token, _ = signTokenPair(ctx, signer, credentials, refreshClaims)

	return result
}
