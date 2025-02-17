package dao_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	pgctx "github.com/a-novel-kit/context/pgbun"

	"github.com/a-novel/authentication/internal/dao"
)

func TestInsertCredentials(t *testing.T) {
	testCases := []struct {
		name string

		fixtures []*dao.CredentialsEntity

		insertData dao.InsertCredentialsData

		expect    *dao.CredentialsEntity
		expectErr error
	}{
		{
			name: "Success",

			insertData: dao.InsertCredentialsData{
				ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Email:    "user@provider.com",
				Password: "password-hashed",
				Now:      time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},

			expect: &dao.CredentialsEntity{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Email:     "user@provider.com",
				Password:  "password-hashed",
				CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "NoPassword",

			insertData: dao.InsertCredentialsData{
				ID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Email: "user@provider.com",
				Now:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},

			expect: &dao.CredentialsEntity{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Email:     "user@provider.com",
				CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "Error/AlreadyExists",

			fixtures: []*dao.CredentialsEntity{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "user@provider.com",
					Password:  "password-2-hashed",
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			insertData: dao.InsertCredentialsData{
				ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Email:    "user@provider.com",
				Password: "password-hashed",
				Now:      time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},

			expectErr: dao.ErrCredentialsAlreadyExists,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			repository := dao.NewInsertCredentialsRepository()

			tx, commit, err := pgctx.NewContextTX(ctx, nil)
			require.NoError(t, err)

			defer func() { _ = commit(false) }()

			db, err := pgctx.Context(tx)
			require.NoError(t, err)

			for _, fixture := range testCase.fixtures {
				_, err = db.NewInsert().Model(fixture).Exec(tx)
				require.NoError(t, err)
			}

			credentials, err := repository.InsertCredentials(tx, testCase.insertData)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, credentials)
		})
	}
}
