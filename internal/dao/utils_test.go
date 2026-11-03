package dao_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-authentication/v2/internal/dao"
)

var (
	// sessionOwnerID is the account the session tests act for.
	sessionOwnerID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	// sessionOtherID is another account, whose sessions the tests leave alone.
	sessionOtherID = uuid.MustParse("00000000-0000-0000-0000-000000000002")
)

// sessionFixtures are the rows a session test starts from. Both accounts always exist.
type sessionFixtures struct {
	sessions    []*dao.Session
	revocations []*dao.RefreshTokenRevocation
}

// insert writes the accounts, then fixtures.
func (fixtures sessionFixtures) insert(ctx context.Context, t *testing.T) {
	t.Helper()

	db, err := postgres.GetContext(ctx)
	require.NoError(t, err)

	created := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	owners := []*dao.Credentials{
		{ID: sessionOwnerID, Email: "owner@provider.com", Role: "auth:user", CreatedAt: created, UpdatedAt: created},
		{ID: sessionOtherID, Email: "other@provider.com", Role: "auth:user", CreatedAt: created, UpdatedAt: created},
	}

	_, err = db.NewInsert().Model(&owners).Exec(ctx)
	require.NoError(t, err)

	if len(fixtures.sessions) > 0 {
		_, err = db.NewInsert().Model(&fixtures.sessions).Exec(ctx)
		require.NoError(t, err)
	}

	if len(fixtures.revocations) > 0 {
		_, err = db.NewInsert().Model(&fixtures.revocations).Exec(ctx)
		require.NoError(t, err)
	}
}

// selectSessions returns every session row, by ID.
func selectSessions(ctx context.Context, t *testing.T) []*dao.Session {
	t.Helper()

	db, err := postgres.GetContext(ctx)
	require.NoError(t, err)

	sessions := make([]*dao.Session, 0)

	require.NoError(t, db.NewSelect().Model(&sessions).Order("id").Scan(ctx))

	return sessions
}

// changed returns a copy of value, changed by update.
func changed[T any](value T, update func(value *T)) *T {
	update(&value)

	return &value
}

// sessionVersion returns the xmin of a session's row: the transaction that wrote its current version.
// Any update of the row changes it.
func sessionVersion(ctx context.Context, t *testing.T, id uuid.UUID) string {
	t.Helper()

	db, err := postgres.GetContext(ctx)
	require.NoError(t, err)

	var version string

	err = db.NewSelect().TableExpr("sessions").ColumnExpr("xmin::text").Where("id = ?", id).Scan(ctx, &version)
	require.NoError(t, err)

	return version
}
