package api_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/internal/api"
	"github.com/a-novel/service-authentication/models/api"
)

func TestPing(t *testing.T) {
	t.Parallel()

	res, err := new(api.API).Ping(t.Context())
	require.NoError(t, err)
	require.Equal(t, &apimodels.PingOK{Data: strings.NewReader("pong")}, res)
}
