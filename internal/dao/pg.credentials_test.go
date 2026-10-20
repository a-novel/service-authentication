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

// The schema refuses a role the application does not know, and defaults new rows to one it does.
func TestCredentials(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		// role is the inserted role; empty leaves the column to its default.
		role string

		expectRole string
		expectErr  bool
	}{
		{
			// 'user' looks like a role but names none the application defines.
			name: "Error/UnknownRole",

			role: "user",

			expectErr: true,
		},
		{
			name: "Success/DefaultRole",

			expectRole: "auth:user",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				db, err := postgres.GetContext(ctx)
				require.NoError(t, err)

				credentials := &dao.Credentials{
					ID:        uuid.New(),
					Email:     "user@email.com",
					Password:  "hash",
					Role:      testCase.role,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}

				insert := db.NewInsert().Model(credentials)
				if testCase.role == "" {
					insert = insert.ExcludeColumn("role")
				}

				_, err = insert.Exec(ctx)
				if testCase.expectErr {
					require.Error(t, err)

					return
				}

				require.NoError(t, err)

				var role string

				require.NoError(t, db.NewRaw("SELECT role FROM credentials WHERE id = ?", credentials.ID).Scan(ctx, &role))
				require.Equal(t, testCase.expectRole, role)
			})
		})
	}
}
