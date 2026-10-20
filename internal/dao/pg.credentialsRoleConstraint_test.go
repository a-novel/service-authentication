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
func TestCredentialsRoleSchema(t *testing.T) {
	t.Parallel()

	t.Run("Error/UnknownRole", func(t *testing.T) {
		t.Parallel()

		postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
			t.Helper()

			db, err := postgres.GetContext(ctx)
			require.NoError(t, err)

			// 'user' looks like a role but names none the application defines.
			credentials := &dao.Credentials{
				ID:        uuid.New(),
				Email:     "bad-role@email.com",
				Role:      "user",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			_, err = db.NewInsert().Model(credentials).Exec(ctx)
			require.Error(t, err, "the database must reject a role the application does not define")
		})
	})

	t.Run("Success/DefaultRole", func(t *testing.T) {
		t.Parallel()

		postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
			t.Helper()

			db, err := postgres.GetContext(ctx)
			require.NoError(t, err)

			id := uuid.New()

			// Insert without naming a role, so the column default applies.
			_, err = db.NewRaw(
				"INSERT INTO credentials (id, email, password, created_at, updated_at) VALUES (?, ?, ?, now(), now())",
				id, "defaulted@email.com", "hash",
			).Exec(ctx)
			require.NoError(t, err)

			var role string

			require.NoError(t, db.NewRaw("SELECT role FROM credentials WHERE id = ?", id).Scan(ctx, &role))
			require.Equal(t, "auth:user", role, "the default must be a role the application knows")
		})
	})
}
