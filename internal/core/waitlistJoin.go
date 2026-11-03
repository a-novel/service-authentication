package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/dao"
)

// WaitlistJoinCredentials checks the authoritative account store before and after a sheet write.
type WaitlistJoinCredentials interface {
	Exec(ctx context.Context, request *dao.CredentialsExistRequest) (bool, error)
}

// WaitlistJoinWriter adds or removes rows through the serialized sheet writer.
type WaitlistJoinWriter interface {
	Exec(ctx context.Context, request *dao.WaitlistRequest) (*dao.WaitlistResult, error)
}

// WaitlistJoinRequest records interest in an invitation, not permission to create an account.
type WaitlistJoinRequest struct {
	Email string `validate:"required,email,max=1024"`
	Lang  string `validate:"required,langs"`
}

// WaitlistJoin adds an address to the invitation list. A registered or already listed address
// succeeds alike and changes nothing, so the answer tells no caller which addresses are known. No
// path sends mail.
type WaitlistJoin struct {
	credentials WaitlistJoinCredentials
	writer      WaitlistJoinWriter
}

// NewWaitlistJoin injects account lookup and sheet storage; no Google credentials enter the core.
func NewWaitlistJoin(credentials WaitlistJoinCredentials, writer WaitlistJoinWriter) *WaitlistJoin {
	return &WaitlistJoin{credentials: credentials, writer: writer}
}

// Exec keeps the sheet free of registered accounts, including a concurrent registration.
func (service *WaitlistJoin) Exec(ctx context.Context, request *WaitlistJoinRequest) error {
	ctx, span := otel.Tracer().Start(ctx, "core.WaitlistJoin")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	lookup := &dao.CredentialsExistRequest{Email: request.Email}

	exists, err := service.credentials.Exec(ctx, lookup)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("check account: %w", err))
	}

	// A registered address joins nothing, and costs what a join does: one writer round trip, a removal
	// that keeps the sheet free of registered accounts, as a concurrent registration below does. A
	// deployment without a writer refuses a join and accepts this removal, but serves no list at all.
	if exists {
		_, err = service.writer.Exec(ctx, &dao.WaitlistRequest{Action: dao.WaitlistActionRemove, Email: request.Email})
		if err != nil {
			return otel.ReportError(span, fmt.Errorf("keep registered account off waitlist: %w", err))
		}

		return nil
	}

	_, err = service.writer.Exec(ctx, &dao.WaitlistRequest{
		Action: dao.WaitlistActionJoin, Email: request.Email, Lang: request.Lang,
	})
	// An address already listed stays as it is, and answers like a new one.
	if err != nil && !errors.Is(err, dao.ErrWaitlistAlreadyJoined) {
		return otel.ReportError(span, fmt.Errorf("join waitlist: %w", err))
	}

	// A concurrently committed account may have removed its row before this join acquired the sheet lock.
	// Only registered accounts are removed; pending invitations have no unsubscribe operation.
	exists, err = service.credentials.Exec(ctx, lookup)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("recheck account: %w", err))
	}

	if exists {
		_, err = service.writer.Exec(ctx, &dao.WaitlistRequest{Action: dao.WaitlistActionRemove, Email: request.Email})
		if err != nil {
			return otel.ReportError(span, fmt.Errorf("remove registered account from waitlist: %w", err))
		}
	}

	return nil
}
