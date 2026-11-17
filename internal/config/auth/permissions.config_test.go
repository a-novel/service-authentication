package auth_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/config/auth"
)

func TestPermissions(t *testing.T) {
	t.Parallel()

	t.Run("Priority", func(t *testing.T) {
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
	})

	t.Run("ByRole", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name  string
			roles map[string]auth.Role

			expect    map[string][]string
			expectErr error
		}{
			{
				// A role grants its own permissions, then each inherited role's, in Inherits order. A
				// role without permissions still maps, so it reads as known.
				name: "Success",
				roles: map[string]auth.Role{
					"base":   {},
					"reader": {Inherits: []string{"base"}, Permissions: []string{"read"}},
					"writer": {Inherits: []string{"reader"}, Permissions: []string{"write"}},
					"admin":  {Inherits: []string{"writer", "reader"}, Permissions: []string{"admin"}},
				},
				expect: map[string][]string{
					"base":   nil,
					"reader": {"read"},
					"writer": {"write", "read"},
					"admin":  {"admin", "write", "read", "read"},
				},
			},
			{
				name: "Error/UnknownInheritedRole",
				roles: map[string]auth.Role{
					"reader": {Inherits: []string{"missing"}},
				},
				expectErr: auth.ErrUnknownRole,
			},
			{
				name: "Error/CircularInheritance",
				roles: map[string]auth.Role{
					"a": {Inherits: []string{"b"}},
					"b": {Inherits: []string{"c"}},
					"c": {Inherits: []string{"a"}},
				},
				expectErr: auth.ErrCircularInheritance,
			},
			{
				name: "Error/SelfInheritance",
				roles: map[string]auth.Role{
					"a": {Inherits: []string{"a"}},
				},
				expectErr: auth.ErrCircularInheritance,
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				t.Parallel()

				byRole, err := auth.Permissions{Roles: testCase.roles}.ByRole()
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, byRole)
			})
		}
	})
}
