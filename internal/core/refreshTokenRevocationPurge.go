package core

import (
	"context"
	"fmt"
	"time"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/dao"
)

// RefreshTokenRevocationPurgeBatch bounds the rows one purge statement deletes, so each statement
// holds its row locks briefly.
const RefreshTokenRevocationPurgeBatch = 1000

// SessionRetention is how long a session's row stays after the session ends, the time a device that
// ended stays listed. A device signed out earlier stays up to the refresh token TTL that JSON Keys
// sets, 7 days, longer. The purge reads the end, which never changes, so a renewal leaves the index
// it scans untouched.
const SessionRetention = 30 * 24 * time.Hour

// RefreshTokenRevocationPurgeDao deletes one batch of expired revocations.
type RefreshTokenRevocationPurgeDao interface {
	Exec(
		ctx context.Context, request *dao.RefreshTokenRevocationPurgeRequest,
	) (*dao.RefreshTokenRevocationPurgeResult, error)
}

// RefreshTokenRevocationPurgeDaoSessions deletes one batch of sessions that ended [SessionRetention] ago.
type RefreshTokenRevocationPurgeDaoSessions interface {
	Exec(ctx context.Context, request *dao.SessionPurgeRequest) (*dao.SessionPurgeResult, error)
}

// RefreshTokenRevocationPurgeResult reports a [RefreshTokenRevocationPurge.Exec] run.
type RefreshTokenRevocationPurgeResult struct {
	// Purged counts the revocations the run deleted.
	Purged int64
	// SessionsPurged counts the sessions the run deleted.
	SessionsPurged int64
}

// RefreshTokenRevocationPurge deletes the revocations whose refresh token the verifier already
// refuses on its own, then the sessions that ended [SessionRetention] ago. A periodic job runs it.
type RefreshTokenRevocationPurge struct {
	dao         RefreshTokenRevocationPurgeDao
	daoSessions RefreshTokenRevocationPurgeDaoSessions
}

func NewRefreshTokenRevocationPurge(
	dao RefreshTokenRevocationPurgeDao, daoSessions RefreshTokenRevocationPurgeDaoSessions,
) *RefreshTokenRevocationPurge {
	return &RefreshTokenRevocationPurge{dao: dao, daoSessions: daoSessions}
}

// Exec deletes batch after batch until one comes back short, for each table in turn. It returns the
// rows deleted so far even on failure, and a later run resumes where it stopped.
func (service *RefreshTokenRevocationPurge) Exec(ctx context.Context) (*RefreshTokenRevocationPurgeResult, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.RefreshTokenRevocationPurge")
	defer span.End()

	result := &RefreshTokenRevocationPurgeResult{}
	revocationsRequest := &dao.RefreshTokenRevocationPurgeRequest{Limit: RefreshTokenRevocationPurgeBatch}

	for {
		batch, err := service.dao.Exec(ctx, revocationsRequest)
		if err != nil {
			return result, otel.ReportError(span, fmt.Errorf("purge revocations: %w", err))
		}

		result.Purged += batch.Purged

		if batch.Purged < RefreshTokenRevocationPurgeBatch {
			break
		}
	}

	sessionsRequest := &dao.SessionPurgeRequest{
		EndedBefore: time.Now().Add(-SessionRetention),
		Limit:       RefreshTokenRevocationPurgeBatch,
	}

	for {
		batch, err := service.daoSessions.Exec(ctx, sessionsRequest)
		if err != nil {
			return result, otel.ReportError(span, fmt.Errorf("purge sessions: %w", err))
		}

		result.SessionsPurged += batch.Purged

		if batch.Purged < RefreshTokenRevocationPurgeBatch {
			return result, nil
		}
	}
}
