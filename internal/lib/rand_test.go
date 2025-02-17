package lib_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/authentication/internal/lib"
)

func TestNewRandomURLString(t *testing.T) {
	t.Parallel()

	str, err := lib.NewRandomURLString(10)
	require.NoError(t, err)
	require.Len(t, str, 10)
	require.Regexp(t, "^[a-zA-Z0-9]{10}$", str)
}
