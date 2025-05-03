package testapiclient_test

import (
	"crypto/rand"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/api/apiclient/testapiclient"
	"github.com/a-novel/service-authentication/models"
)

func TestAuthServer(t *testing.T) {
	t.Parallel()

	// Prevent conflicts with other tests.
	testapiclient.AuthAPIPort = 4123

	testapiclient.InitAuthServer()

	user := rand.Text()
	userAnon := rand.Text()
	userSuperAdmin := rand.Text()

	userID := uuid.New()
	userSuperAdminID := uuid.New()

	// Add user to the pool.
	testapiclient.AddPool(user, &models.AccessTokenClaims{
		UserID: &userID,
		Roles:  []models.Role{models.RoleUser},
	})
	testapiclient.AddPool(userAnon, &models.AccessTokenClaims{})
	testapiclient.AddPool(userSuperAdmin, &models.AccessTokenClaims{
		UserID: &userSuperAdminID,
		Roles:  []models.Role{models.RoleSuperAdmin},
	})

	// Test user authentication.
	t.Run("UserAuthentication", func(t *testing.T) {
		t.Parallel()

		claims, err := testapiclient.Authenticate(t.Context(), user)
		require.NoError(t, err)

		require.Equal(t, &models.AccessTokenClaims{
			UserID: &userID,
			Roles:  []models.Role{models.RoleUser},
		}, claims)
	})

	// Test anonymous user authentication.
	t.Run("AnonymousUserAuthentication", func(t *testing.T) {
		t.Parallel()

		claims, err := testapiclient.Authenticate(t.Context(), userAnon)
		require.NoError(t, err)

		require.Equal(t, &models.AccessTokenClaims{}, claims)
	})

	// Test super admin user authentication.
	t.Run("SuperAdminUserAuthentication", func(t *testing.T) {
		t.Parallel()

		claims, err := testapiclient.Authenticate(t.Context(), userSuperAdmin)
		require.NoError(t, err)

		require.Equal(t, &models.AccessTokenClaims{
			UserID: &userSuperAdminID,
			Roles:  []models.Role{models.RoleSuperAdmin},
		}, claims)
	})
}
