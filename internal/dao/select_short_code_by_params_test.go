package dao_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	pgctx "github.com/a-novel-kit/context/pgbun"

	"github.com/a-novel/authentication/internal/dao"
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

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			repository := dao.NewSelectShortCodeByParamsRepository()

			tx, commit, err := pgctx.NewContextTX(ctx, nil)
			require.NoError(t, err)

			defer func() { _ = commit(false) }()

			db, err := pgctx.Context(tx)
			require.NoError(t, err)

			for _, fixture := range testCase.fixtures {
				_, err = db.NewInsert().Model(fixture).Exec(tx)
				require.NoError(t, err)
			}

			credentials, err := repository.SelectShortCodeByParams(tx, testCase.selectData)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, credentials)
		})
	}
}
