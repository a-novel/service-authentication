package dao_test

import (
	"context"
	"crypto/sha256"
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

	created := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)

	insertFixtures(ctx, t, []*dao.Credentials{
		{ID: sessionOwnerID, Email: "owner@provider.com", Role: "auth:user", CreatedAt: created, UpdatedAt: created},
		{ID: sessionOtherID, Email: "other@provider.com", Role: "auth:user", CreatedAt: created, UpdatedAt: created},
	})
	insertFixtures(ctx, t, fixtures.sessions)
	insertFixtures(ctx, t, fixtures.revocations)
}

// insertFixtures inserts a copy of each row. bun scans each inserted row back into its value, and
// parallel cases share their fixtures, so a test never inserts the originals.
func insertFixtures[T any](ctx context.Context, t *testing.T, rows []*T) {
	t.Helper()

	if len(rows) == 0 {
		return
	}

	copied := make([]*T, len(rows))
	for index, row := range rows {
		duplicate := *row
		copied[index] = &duplicate
	}

	db, err := postgres.GetContext(ctx)
	require.NoError(t, err)

	_, err = db.NewInsert().Model(&copied).Exec(ctx)
	require.NoError(t, err)
}

// newAuthorizationCode returns a code of the session owner for client, expiring at expiresAt. The
// code's ID hashes client, so each client a test names holds a code of its own.
func newAuthorizationCode(client string, expiresAt time.Time) *dao.AuthorizationCode {
	id := sha256.Sum256([]byte(client))

	return &dao.AuthorizationCode{
		ID:                   id[:],
		UserID:               sessionOwnerID,
		ClientID:             client,
		RedirectURI:          "https://studio.example.com/auth/callback",
		RedirectURIInRequest: true,
		CodeChallenge:        "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		CodeChallengeMethod:  "S256",
		SessionID:            "session-1",
		SessionExpiresAt:     expiresAt.Add(time.Hour),
		ExpiresAt:            expiresAt,
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
