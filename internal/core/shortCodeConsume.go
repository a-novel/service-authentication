package core

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/dao"
)

// ShortCodeConsumeDao retires a short code once it has been redeemed, so it
// cannot be used a second time.
type ShortCodeConsumeDao interface {
	Exec(ctx context.Context, request *dao.ShortCodeDeleteRequest) (*dao.ShortCode, error)
}

// ShortCodeConsumeRequest names the code to retire, as [ShortCodeVerify] returned it.
type ShortCodeConsumeRequest struct {
	ID uuid.UUID
}

// ShortCodeConsume retires a short code that [ShortCodeVerify] accepted. Call it
// in the transaction of the write the code authorizes, so a failed write leaves
// the code usable.
type ShortCodeConsume struct {
	dao ShortCodeConsumeDao
}

// NewShortCodeConsume wires the consume service to the DAO that retires stored codes.
func NewShortCodeConsume(dao ShortCodeConsumeDao) *ShortCodeConsume {
	return &ShortCodeConsume{dao: dao}
}

// Exec retires the code and returns it. It returns [ErrShortCodeConsumeInvalid]
// when the code was retired or expired since it was verified, so only one of
// two concurrent redemptions succeeds.
func (service *ShortCodeConsume) Exec(ctx context.Context, request *ShortCodeConsumeRequest) (*ShortCode, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ShortCodeConsume")
	defer span.End()

	span.SetAttributes(attribute.String("shortCode.id", request.ID.String()))

	entity, err := service.dao.Exec(ctx, &dao.ShortCodeDeleteRequest{
		ID:      request.ID,
		Now:     time.Now(),
		Comment: dao.ShortCodeDeleteConsumed,
	})
	if err != nil {
		if errors.Is(err, dao.ErrShortCodeDeleteNotFound) {
			err = errors.Join(err, ErrShortCodeConsumeInvalid)
		}

		return nil, otel.ReportError(span, err)
	}

	return newShortCode(entity, ""), nil
}
