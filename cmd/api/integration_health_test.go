package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/internal/api/apiclient/testapiclient"
)

// STORY: The user can call health apis, and they return a 200 status code.

func TestHealthAPI(t *testing.T) {
	client, _, err := testapiclient.GetServerClient()
	require.NoError(t, err)

	_, err = client.Ping(t.Context())
	require.NoError(t, err)

	_, err = client.Healthcheck(t.Context())
	require.NoError(t, err)
}
