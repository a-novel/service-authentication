package dao_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"
	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-authentication/v2/internal/config/configtest"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/migrations"
)

func TestPgAuthorizationCodeInsert(t *testing.T) {
	t.Parallel()

	hourLater := time.Now().UTC().Round(time.Second).Add(time.Hour)
	code := newAuthorizationCode("studio", hourLater)

	request := &dao.AuthorizationCodeInsertRequest{
		ID:                   code.ID,
		UserID:               code.UserID,
		ClientID:             code.ClientID,
		RedirectURI:          code.RedirectURI,
		RedirectURIInRequest: code.RedirectURIInRequest,
		CodeChallenge:        code.CodeChallenge,
		CodeChallengeMethod:  code.CodeChallengeMethod,
		SessionID:            code.SessionID,
		SessionExpiresAt:     code.SessionExpiresAt,
		ExpiresAt:            code.ExpiresAt,
	}

	testCases := []struct {
		name string

		request *dao.AuthorizationCodeInsertRequest

		expect    *dao.AuthorizationCode
		expectErr bool
	}{
		{
			name: "Success",

			request: request,

			expect: code,
		},
		{
			// A code signs in an account that exists.
			name: "Error/UnknownAccount",

			request: changed(*request, func(request *dao.AuthorizationCodeInsertRequest) {
				request.UserID = uuid.MustParse("00000000-0000-0000-0000-000000000003")
			}),

			expectErr: true,
		},
	}

	daoInsert := dao.NewPgAuthorizationCodeInsert()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				sessionFixtures{}.insert(ctx, t)

				err := daoInsert.Exec(ctx, testCase.request)
				if testCase.expectErr {
					require.Error(t, err)

					return
				}

				require.NoError(t, err)

				db, err := postgres.GetContext(ctx)
				require.NoError(t, err)

				stored := &dao.AuthorizationCode{}

				err = db.NewSelect().Model(stored).Where("id = ?", testCase.request.ID).Scan(ctx)
				require.NoError(t, err)
				require.Equal(t, testCase.expect, stored)
			})
		})
	}
}
