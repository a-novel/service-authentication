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
		{name: "NewRequest"},
		{name: "ExistingAccountDoesNothing", exists: true},
		{name: "ConcurrentRegistrationRemovesRow", created: true},
		{name: "InvalidEmail", email: "invalid", invalid: true, expectErr: core.ErrInvalidRequest},
		{
			name: "OversizedEmail", email: strings.Repeat("a", 1025) + "@example.com",
			invalid: true, expectErr: core.ErrInvalidRequest,
		},
		{name: "InvalidLanguage", lang: "xx", invalid: true, expectErr: core.ErrInvalidRequest},
		{name: "AccountLookupFails", lookupErr: errDatabase, expectErr: errDatabase},
		{name: "WriterUnavailable", joinErr: dao.ErrWaitlistUnavailable, expectErr: dao.ErrWaitlistUnavailable},
		{name: "WriterBusy", joinErr: dao.ErrWaitlistBusy, expectErr: dao.ErrWaitlistBusy},
		{name: "RecheckFails", recheckErr: errDatabase, expectErr: errDatabase},
		{
			name: "ConcurrentCleanupFails", created: true,
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

			credentials := coremocks.NewMockWaitlistJoinCredentials(t)
			writer := coremocks.NewMockWaitlistJoinWriter(t)

			var calls []*mock.Call
			if !testCase.invalid {
				calls = append(calls, credentials.EXPECT().Exec(mock.Anything, &dao.CredentialsExistRequest{Email: email}).
					Return(testCase.exists, testCase.lookupErr).Once())
				if !testCase.exists && testCase.lookupErr == nil {
					calls = append(calls, writer.EXPECT().Exec(mock.Anything, &dao.WaitlistRequest{
						Action: "join", Email: email, Lang: lang,
					}).Return(&dao.WaitlistResult{Status: "accepted"}, testCase.joinErr).Once())
					if testCase.joinErr == nil {
						calls = append(calls, credentials.EXPECT().Exec(mock.Anything, &dao.CredentialsExistRequest{Email: email}).
							Return(testCase.created, testCase.recheckErr).Once())
						if testCase.created && testCase.recheckErr == nil {
							calls = append(calls, writer.EXPECT().Exec(mock.Anything, &dao.WaitlistRequest{
								Action: "remove", Email: email,
							}).Return(&dao.WaitlistResult{Status: "accepted"}, testCase.removeErr).Once())
						}
					}
				}
			}

			mock.InOrder(calls...)

			err := core.NewWaitlistJoin(credentials, writer).Exec(t.Context(), &core.WaitlistJoinRequest{
				Email: email, Lang: lang,
			})
			require.ErrorIs(t, err, testCase.expectErr)
		})
	}
}
