package cmdpkg_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/golib/ogen"

	"github.com/a-novel/service-authentication/models/api"
	"github.com/a-novel/service-authentication/pkg"
)

func testAppAuthAnon(ctx context.Context, t *testing.T, appConfig TestConfig) {
	t.Helper()

	security := pkg.NewBearerSource()
	client, err := pkg.NewAPIClient(ctx, fmt.Sprintf("http://localhost:%v/v1", appConfig.API.Port), security)
	require.NoError(t, err)

	t.Log("CheckSession/Unauthenticated")
	{
		_, err = ogen.MustGetResponse[apimodels.CheckSessionRes, *apimodels.UnauthorizedError](
			client.CheckSession(t.Context()),
		)
		require.NoError(t, err)
	}

	security.SetToken(authAnon(t, appConfig))
	checkSession(t, client)
}
