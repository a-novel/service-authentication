package auth_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/config/auth"
	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"
)

func TestPermissions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		role       string
		expectRank int
		expectErr  error
	}{
		{name: "Success/Anon", role: config.RoleAnon},
		{name: "Success/User", role: config.RoleUser, expectRank: 1},
		{name: "Success/Admin", role: config.RoleAdmin, expectRank: 2},
		{name: "Success/SuperAdmin", role: config.RoleSuperAdmin, expectRank: 3},
		{name: "Error/UnknownRole", role: "unknown", expectErr: config.ErrUnknownRole},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			for _, permissions := range []serviceauthentication.Permissions{
				config.PermissionsConfigDefault,
				auth.PermissionsConfigDefault,
			} {
				rank, err := permissions.Priority(testCase.role)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expectRank, rank)
			}
		})
	}
}
