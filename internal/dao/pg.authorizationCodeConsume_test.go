package dao_test

import (
	"context"
	"crypto/sha256"
	"sync"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"
	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-authentication/v2/internal/config/configtest"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/migrations"
)

func TestPgAuthorizationCodeConsume(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Round(time.Second)
	otherCodeID := sha256.Sum256([]byte("other"))

	code := newAuthorizationCode("studio", now.Add(time.Hour))

	consumedCode := *code
	consumedCode.ConsumedAt = new(now)

	alreadyConsumedCode := *code
	alreadyConsumedCode.ConsumedAt = new(now.Add(-time.Minute))

	testCases := []struct {
		name string

		fixture *dao.AuthorizationCode

		request *dao.AuthorizationCodeConsumeRequest

		expect    *dao.AuthorizationCode
		expectErr error
	}{
		{
			name: "Success",

			fixture: code,

			request: &dao.AuthorizationCodeConsumeRequest{ID: code.ID, Now: now},

			expect: &consumedCode,
		},
		{
			// A code exchanges once: a second consumption finds nothing, and keeps the first instant.
			name: "Error/Consumed",

			fixture: &alreadyConsumedCode,

			request: &dao.AuthorizationCodeConsumeRequest{ID: code.ID, Now: now},

			expectErr: dao.ErrAuthorizationCodeNotFound,
		},
		{
			name: "Error/NotFound",

			fixture: code,

			request: &dao.AuthorizationCodeConsumeRequest{ID: otherCodeID[:], Now: now},

			expectErr: dao.ErrAuthorizationCodeNotFound,
		},
	}

	daoConsume := dao.NewPgAuthorizationCodeConsume()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertAccounts(ctx, t)
				insertFixtures(ctx, t, []*dao.AuthorizationCode{testCase.fixture})

				err := daoConsume.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)

				db, err := postgres.GetContext(ctx)
				require.NoError(t, err)

				stored := &dao.AuthorizationCode{}

				err = db.NewSelect().Model(stored).Where("id = ?", code.ID).Scan(ctx)
				require.NoError(t, err)
				require.Equal(t, lo.CoalesceOrEmpty(testCase.expect, testCase.fixture), stored)
			})
		})
	}

	// Concurrent exchanges of one code race on the same row: one alone consumes it.
	t.Run("Success/Concurrent", func(t *testing.T) {
		t.Parallel()

		postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
			t.Helper()

			insertAccounts(ctx, t)
			insertFixtures(ctx, t, []*dao.AuthorizationCode{code})

			const attempts = 10

			errs := make(chan error, attempts)

			var wg sync.WaitGroup

			for range attempts {
				wg.Go(func() {
					errs <- daoConsume.Exec(ctx, &dao.AuthorizationCodeConsumeRequest{ID: code.ID, Now: now})
				})
			}

			wg.Wait()
			close(errs)

			var consumed int

			for err := range errs {
				if err == nil {
					consumed++

					continue
				}

				require.ErrorIs(t, err, dao.ErrAuthorizationCodeNotFound)
			}

			require.Equal(t, 1, consumed)
		})
	})
}
