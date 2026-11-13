package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/dao"
)

// ShortCodeCancelEmailUpdateDaoSelect loads the caller's active email-change code.
type ShortCodeCancelEmailUpdateDaoSelect interface {
	Exec(ctx context.Context, request *dao.ShortCodeSelectRequest) (*dao.ShortCode, error)
}

// ShortCodeCancelEmailUpdateDao retires the email-change code, so it no longer confirms the change.
type ShortCodeCancelEmailUpdateDao interface {
	Exec(ctx context.Context, request *dao.ShortCodeDeleteRequest) (*dao.ShortCode, error)
}

// ShortCodeCancelEmailUpdateRequest names the user whose pending email change to cancel.
type ShortCodeCancelEmailUpdateRequest struct {
	// UserID is the caller, taken from their verified access token.
	UserID uuid.UUID
}

// ShortCodeCancelEmailUpdate cancels the caller's email change awaiting confirmation. The account keeps
// its current address, and the code mailed for the change no longer confirms it. A caller with no
// change awaiting confirmation gets [ErrShortCodeNotFound], as does one whose change was confirmed,
// replaced or expired during the call.
//
// A change to an address already registered received no mail, and cancels alike, so the call reveals
// nothing about that address.
type ShortCodeCancelEmailUpdate struct {
	dao       ShortCodeCancelEmailUpdateDao
	daoSelect ShortCodeCancelEmailUpdateDaoSelect
}

func NewShortCodeCancelEmailUpdate(
	dao ShortCodeCancelEmailUpdateDao, daoSelect ShortCodeCancelEmailUpdateDaoSelect,
) *ShortCodeCancelEmailUpdate {
	return &ShortCodeCancelEmailUpdate{dao: dao, daoSelect: daoSelect}
}

func (service *ShortCodeCancelEmailUpdate) Exec(ctx context.Context, request *ShortCodeCancelEmailUpdateRequest) error {
	ctx, span := otel.Tracer().Start(ctx, "core.ShortCodeCancelEmailUpdate")
	defer span.End()

	span.SetAttributes(attribute.String("user.id", request.UserID.String()))

	selectRequest := &dao.ShortCodeSelectRequest{
		Usage:  ShortCodeUsageValidateEmail,
		Target: request.UserID.String(),
	}

	shortCode, err := service.daoSelect.Exec(ctx, selectRequest)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("select email update short code: %w", err))
	}

	deleteRequest := &dao.ShortCodeDeleteRequest{
		ID:      shortCode.ID,
		Now:     time.Now(),
		Comment: dao.ShortCodeDeleteCanceled,
	}

	// The delete misses when a concurrent request confirmed or replaced the change, or the code
	// expired, since the select.
	_, err = service.dao.Exec(ctx, deleteRequest)
	if errors.Is(err, dao.ErrShortCodeDeleteNotFound) {
		err = errors.Join(err, ErrShortCodeNotFound)
	}

	if err != nil {
		return otel.ReportError(span, fmt.Errorf("delete email update short code: %w", err))
	}

	return nil
}
