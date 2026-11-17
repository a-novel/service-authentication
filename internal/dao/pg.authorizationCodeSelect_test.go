package dao_test

import (
	"context"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-authentication/v2/internal/config/configtest"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/migrations"
)

func TestPgAuthorizationCodeSelect(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Round(time.Second)
	otherCodeID := sha256.Sum256([]byte("other"))

	code := newAuthorizationCode("studio", now.Add(time.Hour))

	// An exchanged and expired code stays readable, so a second exchange finds it.
	consumedCode := *code
	consumedCode.ExpiresAt = now.Add(-time.Minute)
	consumedCode.ConsumedAt = new(now.Add(-2 * time.Minute))

	testCases := []struct {
		name string

		fixture *dao.AuthorizationCode

		request *dao.AuthorizationCodeSelectRequest

		expect    *dao.AuthorizationCode
		expectErr error
	}{
		{
			name: "Success",

			fixture: code,

			request: &dao.AuthorizationCodeSelectRequest{ID: code.ID},

			expect: code,
		},
		{
			name: "Success/Consumed",

			fixture: &consumedCode,

			request: &dao.AuthorizationCodeSelectRequest{ID: code.ID},

			expect: &consumedCode,
		},
		{
			name: "Error/NotFound",

			fixture: code,

			request: &dao.AuthorizationCodeSelectRequest{ID: otherCodeID[:]},

			expectErr: dao.ErrAuthorizationCodeNotFound,
		},
	}

	daoSelect := dao.NewPgAuthorizationCodeSelect()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertAccounts(ctx, t)
				insertFixtures(ctx, t, []*dao.AuthorizationCode{testCase.fixture})

				res, err := daoSelect.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, res)
			})
		})
	}
}
