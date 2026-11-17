package core_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
)

func TestWaitlistCleanup(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type page struct {
		emails    []string
		accounts  []*dao.Credentials
		listErr   error
		lookupErr error
		removeErr error
		removed   int
	}

	fullPages := make([]page, authconfig.WaitlistMaxEntries/authconfig.WaitlistBatchSize)
	for i := range fullPages {
		fullPages[i].emails = make([]string, authconfig.WaitlistBatchSize)
		for j := range fullPages[i].emails {
			fullPages[i].emails[j] = fmt.Sprintf("%05d@example.com", i*authconfig.WaitlistBatchSize+j)
		}
	}

	testCases := []struct {
		name          string
		apply         bool
		pages         []page
		expect        core.WaitlistCleanupResult
		expectErr     error
		expectMessage string
	}{
		{name: "Success/Empty", pages: []page{{}}},
		{
			name: "Success", pages: []page{
				{emails: []string{"a@example.com", "b@example.com"}, accounts: []*dao.Credentials{{Email: "a@example.com"}}}, {},
			}, expect: core.WaitlistCleanupResult{Scanned: 2, Matched: 1},
		},
		{
			name: "Success/ApplyAcrossPages", apply: true, pages: []page{
				{
					emails:   []string{"a@example.com", "b@example.com"},
					accounts: []*dao.Credentials{{Email: "a@example.com"}}, removed: 2,
				},
				{emails: []string{"c@example.com"}, accounts: []*dao.Credentials{{Email: "c@example.com"}}, removed: 1},
				{},
			}, expect: core.WaitlistCleanupResult{Scanned: 3, Matched: 2, Removed: 3},
		},
		{
			name: "Success/PendingInvitationsRemain", apply: true, pages: []page{{emails: []string{"a@example.com"}}, {}},
			expect: core.WaitlistCleanupResult{Scanned: 1},
		},
		{name: "Error/List", pages: []page{{listErr: errFoo}}, expectErr: errFoo},
		{name: "Error/Cancelled", pages: []page{{listErr: context.Canceled}}, expectErr: context.Canceled},
		{name: "Error/Lookup", pages: []page{{emails: []string{"a@example.com"}, lookupErr: errFoo}}, expectErr: errFoo},
		{
			name: "Error/PartialFailure", apply: true, pages: []page{
				{emails: []string{"a@example.com"}, accounts: []*dao.Credentials{{Email: "a@example.com"}}, removed: 1},
				{emails: []string{"b@example.com"}, accounts: []*dao.Credentials{{Email: "b@example.com"}}, removeErr: errFoo},
			}, expect: core.WaitlistCleanupResult{Scanned: 2, Matched: 2, Removed: 1}, expectErr: errFoo,
		},
		{
			name:   "Error/RepeatedCursor",
			pages:  []page{{emails: []string{"a@example.com"}}, {emails: []string{"a@example.com"}}},
			expect: core.WaitlistCleanupResult{Scanned: 1}, expectErr: core.ErrWaitlistUnavailable,
			expectMessage: "cursor did not advance; check the Apps Script list operation",
		},
		{
			name: "Error/OversizedPage", pages: []page{{emails: make([]string, authconfig.WaitlistBatchSize+1)}},
			expectErr: core.ErrWaitlistUnavailable, expectMessage: "page contains 101 entries; limit is 100",
		},
		{
			name: "Success/ScanLimitReached", pages: append(slices.Clone(fullPages), page{}),
			expect: core.WaitlistCleanupResult{Scanned: authconfig.WaitlistMaxEntries},
		},
		{
			name:   "Error/ScanLimitExceeded",
			pages:  append(slices.Clone(fullPages), page{emails: []string{"overflow@example.com"}}),
			expect: core.WaitlistCleanupResult{Scanned: authconfig.WaitlistMaxEntries}, expectErr: core.ErrWaitlistUnavailable,
			expectMessage: "cleanup would scan 10001 entries; limit is 10000",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			daoCredentialsList := coremocks.NewMockWaitlistCleanupDaoCredentialsList(t)
			daoWaitlist := coremocks.NewMockWaitlistCleanupDaoWaitlist(t)

			var calls []*mock.Call

			after := ""
			scanned := 0

			for _, page := range testCase.pages {
				calls = append(calls, daoWaitlist.EXPECT().Exec(mock.Anything, &dao.WaitlistRequest{Action: "list", After: after}).
					Return(&dao.WaitlistResult{Emails: page.emails}, page.listErr).Once())
				if page.listErr != nil || len(page.emails) == 0 || len(page.emails) > authconfig.WaitlistBatchSize ||
					scanned+len(page.emails) > authconfig.WaitlistMaxEntries {
					break
				}

				next := page.emails[len(page.emails)-1]
				if next == after {
					break
				}

				calls = append(calls, daoCredentialsList.EXPECT().Exec(mock.Anything, &dao.CredentialsListRequest{
					Limit: authconfig.WaitlistBatchSize, Emails: page.emails,
				}).Return(page.accounts, page.lookupErr).Once())
				if testCase.apply && len(page.accounts) > 0 && page.lookupErr == nil {
					emails := make([]string, len(page.accounts))
					for i, account := range page.accounts {
						emails[i] = account.Email
					}

					calls = append(calls, daoWaitlist.EXPECT().
						Exec(mock.Anything, &dao.WaitlistRequest{Action: "remove", Emails: emails}).
						Return(&dao.WaitlistResult{Removed: page.removed}, page.removeErr).Once())
				}

				after = next
				scanned += len(page.emails)
			}

			mock.InOrder(calls...)

			service := core.NewWaitlistCleanup(daoCredentialsList, daoWaitlist)

			result, err := service.Exec(t.Context(), &core.WaitlistCleanupRequest{Apply: testCase.apply})
			require.ErrorIs(t, err, testCase.expectErr)

			if testCase.expectMessage != "" {
				require.ErrorContains(t, err, testCase.expectMessage)
				require.NotContains(t, err.Error(), "@example.com")
			}

			require.Equal(t, &testCase.expect, result)
		})
	}
}
