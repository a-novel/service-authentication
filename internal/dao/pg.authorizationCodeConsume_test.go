package dao_test

import (
	"context"
	"crypto/sha256"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"
	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-authentication/v2/internal/config/configtest"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/migrations"
)

func TestPgAuthorizationCodeConsume(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Round(time.Second)
	codeID := sha256.Sum256([]byte("code-1"))
	otherCodeID := sha256.Sum256([]byte("code-2"))

	account := &dao.Credentials{
		ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Email:     "user@example.com",
		Role:      "auth:user",
		CreatedAt: now,
		UpdatedAt: now,
	}

	code := &dao.AuthorizationCode{
		ID:                   codeID[:],
		UserID:               account.ID,
		ClientID:             "studio",
		RedirectURI:          "https://studio.example.com/auth/callback",
		RedirectURIInRequest: true,
		CodeChallenge:        "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		CodeChallengeMethod:  "S256",
		SessionID:            "session-1",
		SessionExpiresAt:     now.Add(time.Hour),
		ExpiresAt:            now.Add(time.Minute),
	}

	consumedCode := *code
	consumedCode.ConsumedAt = lo.ToPtr(now)

	alreadyConsumedCode := *code
	alreadyConsumedCode.ConsumedAt = lo.ToPtr(now.Add(-time.Minute))

	testCases := []struct {
		name string

		fixture *dao.AuthorizationCode

		request *dao.AuthorizationCodeConsumeRequest

		expect    *dao.AuthorizationCode
		expectErr error
	}{
		{
			name: "Success",

			fixture: code,

			request: &dao.AuthorizationCodeConsumeRequest{ID: codeID[:], Now: now},

			expect: &consumedCode,
		},
		{
			// A code exchanges once: a second consumption finds nothing, and keeps the first instant.
			name: "Error/Consumed",

			fixture: &alreadyConsumedCode,

			request: &dao.AuthorizationCodeConsumeRequest{ID: codeID[:], Now: now},

			expectErr: dao.ErrAuthorizationCodeConsumeNotFound,
		},
		{
			name: "Error/NotFound",

			fixture: code,

			request: &dao.AuthorizationCodeConsumeRequest{ID: otherCodeID[:], Now: now},

			expectErr: dao.ErrAuthorizationCodeConsumeNotFound,
		},
	}

	daoConsume := dao.NewPgAuthorizationCodeConsume()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				db, err := postgres.GetContext(ctx)
				require.NoError(t, err)

				_, err = db.NewInsert().Model(account).Exec(ctx)
				require.NoError(t, err)

				_, err = db.NewInsert().Model(testCase.fixture).Exec(ctx)
				require.NoError(t, err)

				res, err := daoConsume.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, res)

				stored := &dao.AuthorizationCode{}

				err = db.NewSelect().Model(stored).Where("id = ?", codeID[:]).Scan(ctx)
				require.NoError(t, err)
				require.Equal(t, lo.CoalesceOrEmpty(testCase.expect, testCase.fixture), stored)
			})
		})
	}

	// Concurrent exchanges of one code race on the same row: one alone consumes it.
	t.Run("Success/Concurrent", func(t *testing.T) {
		t.Parallel()

		postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
			t.Helper()

			db, err := postgres.GetContext(ctx)
			require.NoError(t, err)

			_, err = db.NewInsert().Model(account).Exec(ctx)
			require.NoError(t, err)

			_, err = db.NewInsert().Model(code).Exec(ctx)
			require.NoError(t, err)

			const attempts = 10

			errs := make(chan error, attempts)

			var wg sync.WaitGroup

			for range attempts {
				wg.Go(func() {
					_, err := daoConsume.Exec(ctx, &dao.AuthorizationCodeConsumeRequest{ID: codeID[:], Now: now})
					errs <- err
				})
			}

			wg.Wait()
			close(errs)

			var consumed int

			for err := range errs {
				if err == nil {
					consumed++

					continue
				}

				require.ErrorIs(t, err, dao.ErrAuthorizationCodeConsumeNotFound)
			}

			require.Equal(t, 1, consumed)
		})
	})
}
