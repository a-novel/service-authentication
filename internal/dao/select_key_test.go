package dao_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	pgctx "github.com/a-novel-kit/context/pgbun"

	"github.com/a-novel/authentication/internal/dao"
	"github.com/a-novel/authentication/models"
)

func TestSelectKey(t *testing.T) {
	hourAgo := time.Now().Add(-time.Hour).UTC().Round(time.Second)
	hourLater := time.Now().Add(time.Hour).UTC().Round(time.Second)

	testCases := []struct {
		name string

		fixtures []*dao.KeyEntity

		id uuid.UUID

		expect    *dao.KeyEntity
		expectErr error
	}{
		{
			name: "Success",

			id: uuid.MustParse("00000000-0000-0000-0000-000000000002"),

			fixtures: []*dao.KeyEntity{
				{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					PrivateKey: "cHJpdmF0ZS1rZXktMQ",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
					Usage:      models.KeyUsageAuth,
					CreatedAt:  hourAgo,
					ExpiresAt:  hourLater,
				},
				{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					PrivateKey: "cHJpdmF0ZS1rZXktMg",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0y"),
					Usage:      models.KeyUsageAuth,
					CreatedAt:  hourAgo,
					ExpiresAt:  hourLater,
				},
			},

			expect: &dao.KeyEntity{
				ID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				PrivateKey: "cHJpdmF0ZS1rZXktMg",
				PublicKey:  lo.ToPtr("cHVibGljLWtleS0y"),
				Usage:      models.KeyUsageAuth,
				CreatedAt:  hourAgo,
				ExpiresAt:  hourLater,
			},
		},
		{
			name: "NotFound",

			id: uuid.MustParse("00000000-0000-0000-0000-000000000002"),

			fixtures: []*dao.KeyEntity{
				{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					PrivateKey: "cHJpdmF0ZS1rZXktMQ",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
					Usage:      models.KeyUsageAuth,
					CreatedAt:  hourAgo,
					ExpiresAt:  hourLater,
				},
			},

			expectErr: dao.ErrKeyNotFound,
		},
		{
			name: "IgnoreExpired",

			id: uuid.MustParse("00000000-0000-0000-0000-000000000002"),

			fixtures: []*dao.KeyEntity{
				{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					PrivateKey: "cHJpdmF0ZS1rZXktMQ",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
					Usage:      models.KeyUsageAuth,
					CreatedAt:  hourAgo,
					ExpiresAt:  hourLater,
				},
				{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					PrivateKey: "cHJpdmF0ZS1rZXktMg",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0y"),
					Usage:      models.KeyUsageAuth,
					CreatedAt:  hourAgo,
					ExpiresAt:  hourAgo,
				},
			},

			expectErr: dao.ErrKeyNotFound,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			selectKey := dao.NewSelectKeyRepository()

			tx, commit, err := pgctx.NewContextTX(ctx, nil)
			require.NoError(t, err)

			defer func() { _ = commit(false) }()

			db, err := pgctx.Context(tx)
			require.NoError(t, err)

			for _, fixture := range testCase.fixtures {
				_, err = db.NewInsert().Model(fixture).Exec(tx)
				require.NoError(t, err)
			}

			key, err := selectKey.SelectKey(tx, testCase.id)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, key)
		})
	}
}
