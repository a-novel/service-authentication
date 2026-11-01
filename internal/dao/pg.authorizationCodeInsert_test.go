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

func TestPgAuthorizationCodeInsert(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Round(time.Second)
	codeID := sha256.Sum256([]byte("code-1"))

	account := &dao.Credentials{
		ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Email:     "user@example.com",
		Role:      "auth:user",
		CreatedAt: now,
		UpdatedAt: now,
	}

	testCases := []struct {
		name string

		request *dao.AuthorizationCodeInsertRequest

		expect    *dao.AuthorizationCode
		expectErr bool
	}{
		{
			name: "Success",

			request: &dao.AuthorizationCodeInsertRequest{
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
			},

			expect: &dao.AuthorizationCode{
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
			},
		},
		{
			// A code signs in an account that exists.
			name: "Error/UnknownAccount",

			request: &dao.AuthorizationCodeInsertRequest{
				ID:                   codeID[:],
				UserID:               uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				ClientID:             "studio",
				RedirectURI:          "https://studio.example.com/auth/callback",
				RedirectURIInRequest: true,
				CodeChallenge:        "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
				CodeChallengeMethod:  "S256",
				SessionID:            "session-1",
				SessionExpiresAt:     now.Add(time.Hour),
				ExpiresAt:            now.Add(time.Minute),
			},

			expectErr: true,
		},
	}

	daoInsert := dao.NewPgAuthorizationCodeInsert()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				db, err := postgres.GetContext(ctx)
				require.NoError(t, err)

				_, err = db.NewInsert().Model(account).Exec(ctx)
				require.NoError(t, err)

				err = daoInsert.Exec(ctx, testCase.request)
				if testCase.expectErr {
					require.Error(t, err)

					return
				}

				require.NoError(t, err)

				stored := &dao.AuthorizationCode{}

				err = db.NewSelect().Model(stored).Where("id = ?", testCase.request.ID).Scan(ctx)
				require.NoError(t, err)
				require.Equal(t, testCase.expect, stored)
			})
		})
	}
}
