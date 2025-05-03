package apiclient_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/api/apiclient"
	"github.com/a-novel/service-authentication/api/codegen"
)

func TestSecuritySource(t *testing.T) {
	t.Parallel()

	securitySource := apiclient.NewSecuritySource()

	require.Empty(t, securitySource.GetToken())

	securitySource.SetToken("foo")

	require.Equal(t, "foo", securitySource.GetToken())

	auth, err := securitySource.BearerAuth(t.Context(), "")
	require.NoError(t, err)
	require.Equal(t, codegen.BearerAuth{Token: "foo"}, auth)
}
