package dao_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-authentication/v2/internal/config/configtest"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/migrations"
)

func TestPgCredentialsLocationConsent(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		// consent is the owner's stored consent.
		consent bool

		request *dao.CredentialsLocationConsentRequest

		expect    bool
		expectErr error
	}{
		{
			name: "Success/On",

			consent: true,

			request: &dao.CredentialsLocationConsentRequest{ID: sessionOwnerID},

			expect: true,
		},
		{
			name: "Success/Off",

			request: &dao.CredentialsLocationConsentRequest{ID: sessionOwnerID},
		},
		{
			name: "Error/NotFound",

			request: &dao.CredentialsLocationConsentRequest{ID: uuid.MustParse("00000000-0000-0000-0000-000000000003")},

			expectErr: dao.ErrCredentialsLocationConsentNotFound,
		},
	}

	daoConsent := dao.NewPgCredentialsLocationConsent()
	daoUpdate := dao.NewPgCredentialsUpdateLocationConsent()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				sessionFixtures{}.insert(ctx, t)

				consentRequest := &dao.CredentialsUpdateLocationConsentRequest{
					ID: sessionOwnerID, LocationConsent: testCase.consent,
				}

				_, err := daoUpdate.Exec(ctx, consentRequest)
				require.NoError(t, err)

				consent, err := daoConsent.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, consent)
			})
		})
	}
}
