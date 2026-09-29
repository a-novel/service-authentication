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

// WaitlistJoin acknowledges existing accounts and duplicate requests without sending mail.
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
	ctx, span := otel.Tracer().Start(ctx, "service.WaitlistJoin")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return otel.ReportError(span, errors.Join(ErrInvalidRequest, err))
	}

	lookup := &dao.CredentialsExistRequest{Email: request.Email}

	exists, err := service.credentials.Exec(ctx, lookup)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("check account: %w", err))
	}

	if exists {
		return nil
	}

	_, err = service.writer.Exec(ctx, &dao.WaitlistRequest{Action: "join", Email: request.Email, Lang: request.Lang})
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("join waitlist: %w", err))
	}

	// Registration may have committed and removed its row before this join acquired the sheet lock.
	exists, err = service.credentials.Exec(ctx, lookup)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("recheck account: %w", err))
	}

	if exists {
		_, err = service.writer.Exec(ctx, &dao.WaitlistRequest{Action: "remove", Email: request.Email})
		if err != nil {
			return otel.ReportError(span, fmt.Errorf("remove registered account from waitlist: %w", err))
		}
	}

	return nil
}
