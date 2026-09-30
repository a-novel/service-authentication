package core_test

import (
	"context"
	"errors"
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

	testCases := []struct {
		name      string
		apply     bool
		pages     []page
		expect    core.WaitlistCleanupResult
		expectErr error
	}{
		{name: "Empty", pages: []page{{}}},
		{
			name: "Preview", pages: []page{
				{emails: []string{"a@example.com", "b@example.com"}, accounts: []*dao.Credentials{{Email: "a@example.com"}}}, {},
			}, expect: core.WaitlistCleanupResult{Scanned: 2, Matched: 1},
		},
		{
			name: "ApplyAcrossPages", apply: true, pages: []page{
				{
					emails:   []string{"a@example.com", "b@example.com"},
					accounts: []*dao.Credentials{{Email: "a@example.com"}}, removed: 2,
				},
				{emails: []string{"c@example.com"}, accounts: []*dao.Credentials{{Email: "c@example.com"}}, removed: 1},
				{},
			}, expect: core.WaitlistCleanupResult{Scanned: 3, Matched: 2, Removed: 3},
		},
		{
			name: "PendingInvitationsRemain", apply: true, pages: []page{{emails: []string{"a@example.com"}}, {}},
			expect: core.WaitlistCleanupResult{Scanned: 1},
		},
		{name: "ListFails", pages: []page{{listErr: errFoo}}, expectErr: errFoo},
		{name: "Cancelled", pages: []page{{listErr: context.Canceled}}, expectErr: context.Canceled},
		{name: "LookupFails", pages: []page{{emails: []string{"a@example.com"}, lookupErr: errFoo}}, expectErr: errFoo},
		{
			name: "PartialFailure", apply: true, pages: []page{
				{emails: []string{"a@example.com"}, accounts: []*dao.Credentials{{Email: "a@example.com"}}, removed: 1},
				{emails: []string{"b@example.com"}, accounts: []*dao.Credentials{{Email: "b@example.com"}}, removeErr: errFoo},
			}, expect: core.WaitlistCleanupResult{Scanned: 2, Matched: 2, Removed: 1}, expectErr: errFoo,
		},
		{
			name: "RepeatedCursor", pages: []page{{emails: []string{"a@example.com"}}, {emails: []string{"a@example.com"}}},
			expect: core.WaitlistCleanupResult{Scanned: 1}, expectErr: core.ErrWaitlistUnavailable,
		},
		{
			name: "OversizedPage", pages: []page{{emails: make([]string, authconfig.WaitlistBatchSize+1)}},
			expectErr: core.ErrWaitlistUnavailable,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			credentials := coremocks.NewMockWaitlistCleanupCredentials(t)
			writer := coremocks.NewMockWaitlistCleanupWriter(t)

			var calls []*mock.Call

			after := ""
			for _, page := range testCase.pages {
				calls = append(calls, writer.EXPECT().Exec(mock.Anything, &dao.WaitlistRequest{Action: "list", After: after}).
					Return(&dao.WaitlistResult{Emails: page.emails}, page.listErr).Once())
				if page.listErr != nil || len(page.emails) == 0 || len(page.emails) > authconfig.WaitlistBatchSize {
					break
				}

				next := page.emails[len(page.emails)-1]
				if next == after {
					break
				}

				calls = append(calls, credentials.EXPECT().Exec(mock.Anything, &dao.CredentialsListRequest{
					Limit: authconfig.WaitlistBatchSize, Emails: page.emails,
				}).Return(page.accounts, page.lookupErr).Once())
				if testCase.apply && len(page.accounts) > 0 && page.lookupErr == nil {
					emails := make([]string, len(page.accounts))
					for i, account := range page.accounts {
						emails[i] = account.Email
					}

					calls = append(calls, writer.EXPECT().Exec(mock.Anything, &dao.WaitlistRequest{Action: "remove", Emails: emails}).
						Return(&dao.WaitlistResult{Removed: page.removed}, page.removeErr).Once())
				}

				after = next
			}

			mock.InOrder(calls...)

			result, err := core.NewWaitlistCleanup(credentials, writer).Exec(t.Context(), &core.WaitlistCleanupRequest{
				Apply: testCase.apply,
			})
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, &testCase.expect, result)
			credentials.AssertExpectations(t)
			writer.AssertExpectations(t)
		})
	}
}
