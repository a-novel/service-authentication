package services_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/jwk"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
)

func testKeySource[Priv, Pub any](
	ctx context.Context, t *testing.T, usage models.KeyUsage,
	privateSource *jwk.Source[Priv], publicSource *jwk.Source[Pub],
) {
	t.Helper()

	searchKeysDAO := dao.NewSearchKeysRepository()
	insertKeyDAO := dao.NewInsertKeyRepository()
	selectKeyDAO := dao.NewSelectKeyRepository()

	generator := services.NewGenerateKeyService(services.NewGenerateKeySource(searchKeysDAO, insertKeyDAO))
	selectKey := services.NewSelectKeyService(selectKeyDAO)

	kid, err := generator.GenerateKey(ctx, usage)
	require.NoError(t, err)
	require.NotNil(t, kid)

	privateKey, err := selectKey.SelectKey(ctx, services.SelectKeyRequest{ID: lo.FromPtr(kid), Private: true})
	require.NoError(t, err)
	publicKey, err := selectKey.SelectKey(ctx, services.SelectKeyRequest{ID: lo.FromPtr(kid)})
	require.NoError(t, err)

	privateKeySourced, err := privateSource.Get(ctx, privateKey.KID)
	require.NoError(t, err)
	publicKeySourced, err := publicSource.Get(ctx, publicKey.KID)
	require.NoError(t, err)

	privateKeyJSON, err := json.Marshal(privateKeySourced)
	require.NoError(t, err)
	publicKeyJSON, err := json.Marshal(publicKeySourced)
	require.NoError(t, err)

	privateKeySourcedJSON, err := json.Marshal(privateKey)
	require.NoError(t, err)
	publicKeySourcedJSON, err := json.Marshal(publicKey)
	require.NoError(t, err)

	require.JSONEq(t, string(privateKeyJSON), string(privateKeySourcedJSON))
	require.JSONEq(t, string(publicKeyJSON), string(publicKeySourcedJSON))
}

func TestKeySources(t *testing.T) { //nolint:paralleltest
	searchKeysDAO := dao.NewSearchKeysRepository()
	searchKeys := services.NewSearchKeysService(searchKeysDAO)

	ctx, err := lib.NewAgoraContext(t.Context())
	require.NoError(t, err)

	t.Run("AuthKeySource", func(t *testing.T) { //nolint:paralleltest
		testKeySource(
			ctx, t, models.KeyUsageAuth,
			services.NewAuthPrivateKeysProvider(searchKeys),
			services.NewAuthPublicKeysProvider(searchKeys),
		)
	})

	t.Run("RefreshKeySource", func(t *testing.T) { //nolint:paralleltest
		testKeySource(
			ctx, t, models.KeyUsageRefresh,
			services.NewRefreshPrivateKeysProvider(searchKeys),
			services.NewRefreshPublicKeysProvider(searchKeys),
		)
	})
}
