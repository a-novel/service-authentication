package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// STORY: The user can call health apis, and they return a 200 status code.

func TestHealthAPI(t *testing.T) {
	t.Parallel()

	client, _, err := getServerClient()
	require.NoError(t, err)

	_, err = client.Ping(t.Context())
	require.NoError(t, err)

	_, err = client.Healthcheck(t.Context())
	require.NoError(t, err)
}
