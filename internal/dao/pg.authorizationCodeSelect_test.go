package dao_test

import (
	"context"
	"crypto/sha256"
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

func TestPgAuthorizationCodeSelect(t *testing.T) {
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

	// An exchanged and expired code stays readable, so a second exchange finds it.
	consumedCode := *code
	consumedCode.ExpiresAt = now.Add(-time.Minute)
	consumedCode.ConsumedAt = lo.ToPtr(now.Add(-2 * time.Minute))

	testCases := []struct {
		name string

		fixture *dao.AuthorizationCode

		request *dao.AuthorizationCodeSelectRequest

		expect    *dao.AuthorizationCode
		expectErr error
	}{
		{
			name: "Success",

			fixture: code,

			request: &dao.AuthorizationCodeSelectRequest{ID: codeID[:]},

			expect: code,
		},
		{
			name: "Success/Consumed",

			fixture: &consumedCode,

			request: &dao.AuthorizationCodeSelectRequest{ID: codeID[:]},

			expect: &consumedCode,
		},
		{
			name: "Error/NotFound",

			fixture: code,

			request: &dao.AuthorizationCodeSelectRequest{ID: otherCodeID[:]},

			expectErr: dao.ErrAuthorizationCodeSelectNotFound,
		},
	}

	daoSelect := dao.NewPgAuthorizationCodeSelect()

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

				res, err := daoSelect.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, res)
			})
		})
	}
}
