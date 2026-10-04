package core

import (
	"context"
	"fmt"

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

		// Enforce the shared limits here as well as in the independently deployed Apps Script.
		if len(page.Emails) > authconfig.WaitlistBatchSize {
			return result, otel.ReportError(span, fmt.Errorf("waitlist page contains %d entries; limit is %d: %w",
				len(page.Emails), authconfig.WaitlistBatchSize, ErrWaitlistUnavailable))
		}

		if result.Scanned+len(page.Emails) > authconfig.WaitlistMaxEntries {
			return result, otel.ReportError(span, fmt.Errorf("waitlist cleanup would scan %d entries; limit is %d: %w",
				result.Scanned+len(page.Emails), authconfig.WaitlistMaxEntries, ErrWaitlistUnavailable))
		}

		next := page.Emails[len(page.Emails)-1]
		if next == after {
			// Stop a stalled writer before processing the same page twice. Cursors contain private emails.
			return result, otel.ReportError(span, fmt.Errorf(
				"waitlist cleanup cursor did not advance; check the Apps Script list operation: %w", ErrWaitlistUnavailable))
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
			emails := make([]string, len(accounts))
			for i, account := range accounts {
				emails[i] = account.Email
			}

			removed, removeErr := service.writer.Exec(ctx, &dao.WaitlistRequest{
				Action: dao.WaitlistActionRemove, Emails: emails,
			})
			if removeErr != nil {
				return result, otel.ReportError(span, fmt.Errorf("remove registered accounts: %w", removeErr))
			}

			result.Removed += removed.Removed
		}

		// Email cursors remain stable when deletion shifts the sheet's row numbers.
		after = next
	}
}
