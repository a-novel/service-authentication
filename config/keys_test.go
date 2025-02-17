package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/authentication/config"
	"github.com/a-novel/authentication/models"
)

func TestKeys(t *testing.T) {
	t.Parallel()

	for _, usage := range models.KnownKeyUsages {
		// Configuration exists (no non-configured usage).
		keyConfig, ok := config.Keys.Usages[usage]
		require.True(t, ok, "key usage %s is not configured", usage)

		// Each field is defined.
		require.NotZero(t, keyConfig.TTL, "key usage %s has no TTL", usage)
		require.NotZero(t, keyConfig.Rotation, "key usage %s has no Rotation", usage)

		// Difference between TTL and Rotation must be significant.
		require.Greater(
			t, keyConfig.TTL, 2*keyConfig.Rotation,
			"key usage %s has a too short TTL (%s) compared to Rotation (%s): must be at least 2 times greater",
			usage, keyConfig.TTL, keyConfig.Rotation,
		)
	}
}
