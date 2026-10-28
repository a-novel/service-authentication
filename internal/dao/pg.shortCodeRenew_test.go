package dao_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"
	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-authentication/v2/internal/config/configtest"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/migrations"
)

func TestPgShortCodeRenew(t *testing.T) {
	t.Parallel()

	twoHoursAgo := time.Now().Add(-2 * time.Hour).UTC().Round(time.Second)
	hourAgo := time.Now().Add(-time.Hour).UTC().Round(time.Second)
	now := time.Now().UTC().Round(time.Second)
	hourLater := time.Now().Add(time.Hour).UTC().Round(time.Second)

	shortCodeID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	request := &dao.ShortCodeRenewRequest{ID: shortCodeID, Code: "new-code", Now: now, ExpiresAt: hourLater}
	renewed := &dao.ShortCode{
		ID:        shortCodeID,
		Code:      "new-code",
		Usage:     "test-usage",
		Target:    "test-target",
		Data:      []byte("test-data"),
		CreatedAt: now,
		ExpiresAt: hourLater,
	}

	testCases := []struct {
		name string

		fixture *dao.ShortCode
		request *dao.ShortCodeRenewRequest

		expect    *dao.ShortCode
		expectErr error
	}{
		{
			name: "Success",

			fixture: &dao.ShortCode{
				ID:        shortCodeID,
				Code:      "old-code",
				Usage:     "test-usage",
				Target:    "test-target",
				Data:      []byte("test-data"),
				CreatedAt: hourAgo,
				ExpiresAt: hourLater,
			},
			request: request,

			expect: renewed,
		},
		{
			// The outbox's deadline, not the code's, stops the retries, so an expired code renews.
			name: "Success/Expired",

			fixture: &dao.ShortCode{
				ID:        shortCodeID,
				Code:      "old-code",
				Usage:     "test-usage",
				Target:    "test-target",
				Data:      []byte("test-data"),
				CreatedAt: twoHoursAgo,
				ExpiresAt: hourAgo,
			},
			request: request,

			expect: renewed,
		},
		{
			// A consumed or superseded code has nothing left to renew.
			name: "Error/Deleted",

			fixture: &dao.ShortCode{
				ID:             shortCodeID,
				Code:           "old-code",
				Usage:          "test-usage",
				Target:         "test-target",
				Data:           []byte("test-data"),
				CreatedAt:      hourAgo,
				ExpiresAt:      hourLater,
				DeletedAt:      &hourAgo,
				DeletedComment: lo.ToPtr(dao.ShortCodeDeleteOverride),
			},
			request: request,

			expectErr: dao.ErrShortCodeRenewNotFound,
		},
		{
			name: "Error/NotFound",

			fixture: &dao.ShortCode{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Code:      "old-code",
				Usage:     "test-usage",
				Target:    "test-target",
				CreatedAt: hourAgo,
				ExpiresAt: hourLater,
			},
			request: request,

			expectErr: dao.ErrShortCodeRenewNotFound,
		},
	}

	renew := dao.NewPgShortCodeRenew()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				db, err := postgres.GetContext(ctx)
				require.NoError(t, err)

				_, err = db.NewInsert().Model(testCase.fixture).Exec(ctx)
				require.NoError(t, err)

				shortCode, err := renew.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, shortCode)
			})
		})
	}
}
