package dao_test

import (
	"context"
	"crypto/sha256"
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

func TestPgAuthorizationCodePurge(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Round(time.Second)

	account := &dao.Credentials{
		ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Email:     "user@example.com",
		Role:      "auth:user",
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Two expired codes, the older first, and one still live. Each code's client names it.
	fixtures := []*dao.AuthorizationCode{
		newPurgeFixture("code-1", account.ID, now.Add(-2*time.Minute)),
		newPurgeFixture("code-2", account.ID, now.Add(-time.Minute)),
		newPurgeFixture("code-3", account.ID, now.Add(time.Minute)),
	}

	testCases := []struct {
		name string

		fixtures []*dao.AuthorizationCode

		request *dao.AuthorizationCodePurgeRequest

		expect *dao.AuthorizationCodePurgeResult
		// expectRemaining lists the clients of the codes left after the call.
		expectRemaining []string
	}{
		{
			name: "Success",

			fixtures: fixtures,

			request: &dao.AuthorizationCodePurgeRequest{Limit: 10},

			expect:          &dao.AuthorizationCodePurgeResult{Purged: 2},
			expectRemaining: []string{"code-3"},
		},
		{
			name: "Success/Limit",

			fixtures: fixtures,

			request: &dao.AuthorizationCodePurgeRequest{Limit: 1},

			expect:          &dao.AuthorizationCodePurgeResult{Purged: 1},
			expectRemaining: []string{"code-2", "code-3"},
		},
		{
			name: "Success/Empty",

			request: &dao.AuthorizationCodePurgeRequest{Limit: 10},

			expect:          &dao.AuthorizationCodePurgeResult{Purged: 0},
			expectRemaining: []string{},
		},
	}

	daoPurge := dao.NewPgAuthorizationCodePurge()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				db, err := postgres.GetContext(ctx)
				require.NoError(t, err)

				_, err = db.NewInsert().Model(account).Exec(ctx)
				require.NoError(t, err)

				if len(testCase.fixtures) > 0 {
					_, err = db.NewInsert().Model(&testCase.fixtures).Exec(ctx)
					require.NoError(t, err)
				}

				res, err := daoPurge.Exec(ctx, testCase.request)
				require.NoError(t, err)
				require.Equal(t, testCase.expect, res)

				remaining := make([]string, 0)

				err = db.NewSelect().
					Model((*dao.AuthorizationCode)(nil)).
					Column("client_id").
					Order("client_id").
					Scan(ctx, &remaining)
				require.NoError(t, err)
				require.Equal(t, testCase.expectRemaining, remaining)
			})
		})
	}
}

// newPurgeFixture returns a code of userID that expires at expiresAt, and whose client is named
// after it.
func newPurgeFixture(name string, userID uuid.UUID, expiresAt time.Time) *dao.AuthorizationCode {
	id := sha256.Sum256([]byte(name))

	return &dao.AuthorizationCode{
		ID:                   id[:],
		UserID:               userID,
		ClientID:             name,
		RedirectURI:          "https://studio.example.com/auth/callback",
		RedirectURIInRequest: true,
		CodeChallenge:        "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		CodeChallengeMethod:  "S256",
		SessionID:            "session-1",
		SessionExpiresAt:     expiresAt.Add(time.Hour),
		ExpiresAt:            expiresAt,
	}
}
