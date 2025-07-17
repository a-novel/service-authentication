package dao_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/golib/postgres"

	"github.com/a-novel/service-authentication/internal/dao"
	testutils "github.com/a-novel/service-authentication/pkg/cmd"
)

func TestSelectShortCodeByParams(t *testing.T) {
	hourAgo := time.Now().Add(-time.Hour).UTC().Round(time.Second)
	hourLater := time.Now().Add(time.Hour).UTC().Round(time.Second)

	testCases := []struct {
		name string

		fixtures []*dao.ShortCodeEntity

		selectData dao.SelectShortCodeByParamsData

		expect    *dao.ShortCodeEntity
		expectErr error
	}{
		{
			name: "Success",

			fixtures: []*dao.ShortCodeEntity{
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

			selectData: dao.SelectShortCodeByParamsData{
				Target: "test-target",
				Usage:  "test-usage",
			},

			expect: &dao.ShortCodeEntity{
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

			selectData: dao.SelectShortCodeByParamsData{
				Target: "test-target",
				Usage:  "test-usage",
			},

			expectErr: dao.ErrShortCodeNotFound,
		},
		{
			name: "IgnoreExpired",

			fixtures: []*dao.ShortCodeEntity{
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

			selectData: dao.SelectShortCodeByParamsData{
				Target: "test-target",
				Usage:  "test-usage",
			},

			expectErr: dao.ErrShortCodeNotFound,
		},
		{
			name: "IgnoreDeleted",

			fixtures: []*dao.ShortCodeEntity{
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

			selectData: dao.SelectShortCodeByParamsData{
				Target: "test-target",
				Usage:  "test-usage",
			},

			expectErr: dao.ErrShortCodeNotFound,
		},
	}

	repository := dao.NewSelectShortCodeByParamsRepository()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			postgres.RunTransactionalTest(t, testutils.PostgresConfigTest, func(ctx context.Context, t *testing.T) {
				t.Helper()

				db, err := postgres.GetContext(ctx)
				require.NoError(t, err)

				if len(testCase.fixtures) > 0 {
					_, err = db.NewInsert().Model(&testCase.fixtures).Exec(ctx)
					require.NoError(t, err)
				}

				credentials, err := repository.SelectShortCodeByParams(ctx, testCase.selectData)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, credentials)
			})
		})
	}
}
