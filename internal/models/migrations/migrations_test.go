package migrations_test

import (
	"io/fs"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-authentication/v2/internal/config/configtest"
	"github.com/a-novel/service-authentication/v2/internal/models/migrations"
)

func TestMigrationRoundtrip(t *testing.T) {
	t.Parallel()

	_, err := fs.Stat(migrations.Migrations, "testdata")
	require.ErrorIs(t, err, fs.ErrNotExist)

	postgres.RunMigrationRoundtripTest(t, configtest.PostgresPreset, migrations.Migrations,
		&postgres.RoundtripOptions{
			Fixtures:  os.DirFS("testdata/fixtures"),
			Snapshots: "testdata/schema",
		})
}
