package auth_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/config/auth"
)

func TestPermissions(t *testing.T) {
	t.Parallel()

	const registrationPermission = "shortCode:register"

	testCases := []struct {
		name                         string
		role                         string
		expectRank                   int
		expectRegistrationPermission bool
		expectErr                    error
	}{
		{name: "Success/Anon", role: auth.RoleAnon},
		{name: "Success/User", role: auth.RoleUser, expectRank: 1},
		{name: "Success/Admin", role: auth.RoleAdmin, expectRank: 2},
		{
			name:                         "Success/SuperAdmin",
			role:                         auth.RoleSuperAdmin,
			expectRank:                   3,
			expectRegistrationPermission: true,
		},
		{name: "Error/UnknownRole", role: "unknown", expectErr: auth.ErrUnknownRole},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			rank, err := auth.PermissionsConfigDefault.Priority(testCase.role)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expectRank, rank)

			if testCase.expectErr == nil {
				require.Equal(
					t,
					testCase.expectRegistrationPermission,
					slices.Contains(auth.PermissionsConfigDefault.Roles[testCase.role].Permissions, registrationPermission),
				)
			}
		})
	}
}
