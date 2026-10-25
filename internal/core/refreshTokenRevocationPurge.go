package core

import (
	"context"
	"fmt"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/dao"
)

// RefreshTokenRevocationPurgeBatch bounds the rows one purge statement deletes, so each statement
// holds its row locks briefly.
const RefreshTokenRevocationPurgeBatch = 1000

// RefreshTokenRevocationPurgeDao deletes one batch of expired revocations.
type RefreshTokenRevocationPurgeDao interface {
	Exec(
		ctx context.Context, request *dao.RefreshTokenRevocationPurgeRequest,
	) (*dao.RefreshTokenRevocationPurgeResult, error)
}

// RefreshTokenRevocationPurgeResult reports a [RefreshTokenRevocationPurge.Exec] run.
type RefreshTokenRevocationPurgeResult struct {
	// Purged counts the revocations the run deleted.
	Purged int64
}

// RefreshTokenRevocationPurge deletes the revocations whose refresh token the verifier already
// refuses on its own. A periodic job runs it.
type RefreshTokenRevocationPurge struct {
	dao RefreshTokenRevocationPurgeDao
}

func NewRefreshTokenRevocationPurge(dao RefreshTokenRevocationPurgeDao) *RefreshTokenRevocationPurge {
	return &RefreshTokenRevocationPurge{dao: dao}
}

// Exec deletes batch after batch until one comes back short. It returns the rows deleted so far
// even on failure, and a later run resumes where it stopped.
func (service *RefreshTokenRevocationPurge) Exec(ctx context.Context) (*RefreshTokenRevocationPurgeResult, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.RefreshTokenRevocationPurge")
	defer span.End()

	result := &RefreshTokenRevocationPurgeResult{}
	request := &dao.RefreshTokenRevocationPurgeRequest{Limit: RefreshTokenRevocationPurgeBatch}

	for {
		batch, err := service.dao.Exec(ctx, request)
		if err != nil {
			return result, otel.ReportError(span, fmt.Errorf("purge revocations: %w", err))
		}

		result.Purged += batch.Purged

		if batch.Purged < RefreshTokenRevocationPurgeBatch {
			return result, nil
		}
	}
}
