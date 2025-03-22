package api_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/authentication/api"
	"github.com/a-novel/authentication/api/codegen"
	"github.com/a-novel/authentication/internal/lib"
)

func TestPing(t *testing.T) {
	t.Parallel()

	res, err := new(api.API).Ping(t.Context())
	require.NoError(t, err)
	require.Equal(t, &codegen.PingOK{Data: strings.NewReader("pong")}, res)
}

func TestHealthcheck(t *testing.T) {
	t.Parallel()

	ctx, err := lib.NewAgoraContext(t.Context())
	require.NoError(t, err)

	res, err := new(api.API).Healthcheck(ctx)
	require.NoError(t, err)

	require.Equal(t, &codegen.Health{
		Postgres: codegen.Dependency{
			Name:   "postgres",
			Status: codegen.DependencyStatusUp,
		},
	}, res)
}
