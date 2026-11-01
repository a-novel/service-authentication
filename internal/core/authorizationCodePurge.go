package core

import (
	"context"
	"fmt"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/dao"
)

// AuthorizationCodePurgeBatch bounds the rows one purge statement deletes, so each statement holds
// its row locks briefly.
const AuthorizationCodePurgeBatch = 1000

// AuthorizationCodePurgeDao deletes one batch of expired codes.
type AuthorizationCodePurgeDao interface {
	Exec(ctx context.Context, request *dao.AuthorizationCodePurgeRequest) (*dao.AuthorizationCodePurgeResult, error)
}

// AuthorizationCodePurgeResult reports a [AuthorizationCodePurge.Exec] run.
type AuthorizationCodePurgeResult struct {
	// Purged counts the codes the run deleted.
	Purged int64
}

// AuthorizationCodePurge deletes the authorization codes that expired. An exchanged code stays until
// then, so a second exchange finds it. A periodic job runs it.
type AuthorizationCodePurge struct {
	dao AuthorizationCodePurgeDao
}

func NewAuthorizationCodePurge(dao AuthorizationCodePurgeDao) *AuthorizationCodePurge {
	return &AuthorizationCodePurge{dao: dao}
}

// Exec deletes batch after batch until one comes back short. It returns the rows deleted so far
// even on failure, and a later run resumes where it stopped.
func (service *AuthorizationCodePurge) Exec(ctx context.Context) (*AuthorizationCodePurgeResult, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.AuthorizationCodePurge")
	defer span.End()

	result := &AuthorizationCodePurgeResult{}
	request := &dao.AuthorizationCodePurgeRequest{Limit: AuthorizationCodePurgeBatch}

	for {
		batch, err := service.dao.Exec(ctx, request)
		if err != nil {
			return result, otel.ReportError(span, fmt.Errorf("purge authorization codes: %w", err))
		}

		result.Purged += batch.Purged

		if batch.Purged < AuthorizationCodePurgeBatch {
			return result, nil
		}
	}
}
