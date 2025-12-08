package dao_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/dao"
)

func TestShortCodeSelect(t *testing.T) {
	hourAgo := time.Now().Add(-time.Hour).UTC().Round(time.Second)
	hourLater := time.Now().Add(time.Hour).UTC().Round(time.Second)

	testCases := []struct {
		name string

		fixtures []*dao.ShortCode

		request *dao.ShortCodeSelectRequest

		expect    *dao.ShortCode
		expectErr error
	}{
		{
			name: "Success",

			fixtures: []*dao.ShortCode{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Code:      "test-code",
					Usage:     "test-usage",
					Target:    "test-target",
					Data:      []byte("test-data"),
					CreatedAt: hourAgo,
					ExpiresAt: hourLater,
				},
			},

			request: &dao.ShortCodeSelectRequest{
				Target: "test-target",
				Usage:  "test-usage",
			},

			expect: &dao.ShortCode{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Code:      "test-code",
				Usage:     "test-usage",
				Target:    "test-target",
				Data:      []byte("test-data"),
				CreatedAt: hourAgo,
				ExpiresAt: hourLater,
			},
		},
		{
			name: "NotFound",

			request: &dao.ShortCodeSelectRequest{
				Target: "test-target",
				Usage:  "test-usage",
			},

			expectErr: dao.ErrShortCodeSelectNotFound,
		},
		{
			name: "IgnoreExpired",

			fixtures: []*dao.ShortCode{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Code:      "test-code",
					Usage:     "test-usage",
					Target:    "test-target",
					Data:      []byte("test-data"),
					CreatedAt: hourAgo,
					ExpiresAt: hourAgo,
				},
			},

			request: &dao.ShortCodeSelectRequest{
				Target: "test-target",
				Usage:  "test-usage",
			},

			expectErr: dao.ErrShortCodeSelectNotFound,
		},
		{
			name: "IgnoreDeleted",

			fixtures: []*dao.ShortCode{
				{
					ID:             uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Code:           "test-code",
					Usage:          "test-usage",
					Target:         "test-target",
					Data:           []byte("test-data"),
					CreatedAt:      hourAgo,
					ExpiresAt:      hourLater,
					DeletedAt:      &hourAgo,
					DeletedComment: lo.ToPtr("test-comment"),
				},
			},

			request: &dao.ShortCodeSelectRequest{
				Target: "test-target",
				Usage:  "test-usage",
			},

			expectErr: dao.ErrShortCodeSelectNotFound,
		},
	}

	repository := dao.NewShortCodeSelect()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			postgres.RunTransactionalTest(t, config.PostgresPresetTest, func(ctx context.Context, t *testing.T) {
				t.Helper()

				db, err := postgres.GetContext(ctx)
				require.NoError(t, err)

				if len(testCase.fixtures) > 0 {
					_, err = db.NewInsert().Model(&testCase.fixtures).Exec(ctx)
					require.NoError(t, err)
				}

				credentials, err := repository.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, credentials)
			})
		})
	}
}
