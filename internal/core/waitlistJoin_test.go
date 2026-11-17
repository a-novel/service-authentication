package core_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
)

func TestWaitlistJoin(t *testing.T) {
	t.Parallel()

	errDatabase := errors.New("database unavailable")

	testCases := []struct {
		name       string
		email      string
		lang       string
		invalid    bool
		exists     bool
		lookupErr  error
		joinErr    error
		created    bool
		recheckErr error
		removeErr  error
		expectErr  error
	}{
		{name: "Success"},
		// A registered address answers like a new one, and its writer round trip adds no row.
		{name: "Success/ExistingAccount", exists: true},
		{
			name: "Error/ExistingAccountWriterUnavailable", exists: true,
			removeErr: dao.ErrWaitlistUnavailable, expectErr: dao.ErrWaitlistUnavailable,
		},
		{name: "Success/Duplicate", joinErr: dao.ErrWaitlistAlreadyJoined},
		{name: "Success/ConcurrentRegistration", created: true},
		{name: "Success/ConcurrentRegistrationWithDuplicate", created: true, joinErr: dao.ErrWaitlistAlreadyJoined},
		{name: "Error/InvalidEmail", email: "invalid", invalid: true, expectErr: core.ErrInvalidRequest},
		{
			name: "Error/OversizedEmail", email: strings.Repeat("a", 1025) + "@example.com",
			invalid: true, expectErr: core.ErrInvalidRequest,
		},
		{name: "Error/InvalidLanguage", lang: "xx", invalid: true, expectErr: core.ErrInvalidRequest},
		{name: "Error/AccountLookup", lookupErr: errDatabase, expectErr: errDatabase},
		{name: "Error/WriterUnavailable", joinErr: dao.ErrWaitlistUnavailable, expectErr: dao.ErrWaitlistUnavailable},
		{name: "Error/WriterBusy", joinErr: dao.ErrWaitlistBusy, expectErr: dao.ErrWaitlistBusy},
		{name: "Error/Recheck", recheckErr: errDatabase, expectErr: errDatabase},
		{
			name: "Error/ConcurrentCleanup", created: true,
			removeErr: dao.ErrWaitlistUnavailable, expectErr: dao.ErrWaitlistUnavailable,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			email, lang := testCase.email, testCase.lang
			if email == "" {
				email = "Member@example.com"
			}

			if lang == "" {
				lang = "fr"
			}

			daoCredentialsExist := coremocks.NewMockWaitlistJoinDaoCredentialsExist(t)
			daoWaitlist := coremocks.NewMockWaitlistJoinDaoWaitlist(t)

			var calls []*mock.Call
			if !testCase.invalid {
				calls = append(calls, daoCredentialsExist.EXPECT().Exec(mock.Anything, &dao.CredentialsExistRequest{Email: email}).
					Return(testCase.exists, testCase.lookupErr).Once())
				if testCase.exists {
					calls = append(calls, daoWaitlist.EXPECT().Exec(mock.Anything, &dao.WaitlistRequest{
						Action: "remove", Email: email,
					}).Return(&dao.WaitlistResult{}, testCase.removeErr).Once())
				}

				if !testCase.exists && testCase.lookupErr == nil {
					calls = append(calls, daoWaitlist.EXPECT().Exec(mock.Anything, &dao.WaitlistRequest{
						Action: "join", Email: email, Lang: lang,
					}).Return(&dao.WaitlistResult{}, testCase.joinErr).Once())
					if testCase.joinErr == nil || errors.Is(testCase.joinErr, dao.ErrWaitlistAlreadyJoined) {
						calls = append(calls, daoCredentialsExist.EXPECT().
							Exec(mock.Anything, &dao.CredentialsExistRequest{Email: email}).
							Return(testCase.created, testCase.recheckErr).Once())
						if testCase.created && testCase.recheckErr == nil {
							calls = append(calls, daoWaitlist.EXPECT().Exec(mock.Anything, &dao.WaitlistRequest{
								Action: "remove", Email: email,
							}).Return(&dao.WaitlistResult{}, testCase.removeErr).Once())
						}
					}
				}
			}

			mock.InOrder(calls...)

			err := core.NewWaitlistJoin(daoCredentialsExist, daoWaitlist).Exec(t.Context(), &core.WaitlistJoinRequest{
				Email: email, Lang: lang,
			})
			require.ErrorIs(t, err, testCase.expectErr)
		})
	}
}
