package core

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/otel"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/dao"
)

// WaitlistCleanupCredentials finds registered accounts within a page of pending addresses.
type WaitlistCleanupCredentials interface {
	Exec(ctx context.Context, request *dao.CredentialsListRequest) ([]*dao.Credentials, error)
}

// WaitlistCleanupWriter reads and removes bounded pages through the signed sheet endpoint.
type WaitlistCleanupWriter interface {
	Exec(ctx context.Context, request *dao.WaitlistRequest) (*dao.WaitlistResult, error)
}

// WaitlistCleanupRequest previews stale entries unless Apply explicitly enables deletion.
type WaitlistCleanupRequest struct {
	// Apply removes addresses that PostgreSQL reports as registered during the scan.
	Apply bool
}

// WaitlistCleanupResult reports aggregate progress without disclosing email addresses.
type WaitlistCleanupResult struct {
	// Scanned counts unique addresses examined.
	Scanned int
	// Matched counts addresses belonging to registered accounts.
	Matched int
	// Removed counts deleted rows, including duplicates; zero during a preview.
	Removed int
}

// WaitlistCleanup reconciles stale sheet entries against PostgreSQL without sending invitations.
type WaitlistCleanup struct {
	credentials WaitlistCleanupCredentials
	writer      WaitlistCleanupWriter
}

// NewWaitlistCleanup injects account lookup and private sheet access.
func NewWaitlistCleanup(credentials WaitlistCleanupCredentials, writer WaitlistCleanupWriter) *WaitlistCleanup {
	return &WaitlistCleanup{credentials: credentials, writer: writer}
}

// Exec returns progress even on failure. Re-running safely rechecks the remaining rows.
// The scan is bounded but not a snapshot: concurrent new entries before its cursor await the next run.
func (service *WaitlistCleanup) Exec(
	ctx context.Context, request *WaitlistCleanupRequest,
) (*WaitlistCleanupResult, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.WaitlistCleanup")
	defer span.End()

	result := &WaitlistCleanupResult{}

	after := ""
	for {
		page, err := service.writer.Exec(ctx, &dao.WaitlistRequest{Action: dao.WaitlistActionList, After: after})
		if err != nil {
			return result, otel.ReportError(span, fmt.Errorf("list waitlist: %w", err))
		}

		if len(page.Emails) == 0 {
			return result, nil
		}

		err = checkWaitlistPage(page.Emails, result.Scanned, after)
		if err != nil {
			return result, otel.ReportError(span, err)
		}

		accounts, err := service.credentials.Exec(ctx, &dao.CredentialsListRequest{
			Limit: authconfig.WaitlistBatchSize, Emails: page.Emails,
		})
		if err != nil {
			return result, otel.ReportError(span, fmt.Errorf("match registered accounts: %w", err))
		}

		result.Scanned += len(page.Emails)
		result.Matched += len(accounts)

		// PostgreSQL decides which rows are stale; pending invitations remain in the sheet.
		if request.Apply && len(accounts) > 0 {
			removed, removeErr := service.writer.Exec(ctx, &dao.WaitlistRequest{
				Action: dao.WaitlistActionRemove, Emails: lo.Map(accounts, credentialsEmail),
			})
			if removeErr != nil {
				return result, otel.ReportError(span, fmt.Errorf("remove registered accounts: %w", removeErr))
			}

			result.Removed += removed.Removed
		}

		// Email cursors remain stable when deletion shifts the sheet's row numbers.
		after = page.Emails[len(page.Emails)-1]
	}
}

// checkWaitlistPage enforces the shared limits on a listed page, as the independently deployed Apps
// Script does, and stops a stalled writer before the same page is processed twice. The errors leave
// out the cursor, which is a private email.
func checkWaitlistPage(emails []string, scanned int, after string) error {
	switch {
	case len(emails) > authconfig.WaitlistBatchSize:
		return fmt.Errorf("waitlist page contains %d entries; limit is %d: %w",
			len(emails), authconfig.WaitlistBatchSize, ErrWaitlistUnavailable)
	case scanned+len(emails) > authconfig.WaitlistMaxEntries:
		return fmt.Errorf("waitlist cleanup would scan %d entries; limit is %d: %w",
			scanned+len(emails), authconfig.WaitlistMaxEntries, ErrWaitlistUnavailable)
	case emails[len(emails)-1] == after:
		return fmt.Errorf("waitlist cleanup cursor did not advance; check the Apps Script list operation: %w",
			ErrWaitlistUnavailable)
	}

	return nil
}

func credentialsEmail(credentials *dao.Credentials, _ int) string {
	return credentials.Email
}
