package lib_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/authentication/internal/lib"
)

func TestMasterKeyContext(t *testing.T) {
	testCases := []struct {
		name string

		envValue string

		expect    [32]byte
		expectErr bool
	}{
		{
			name: "LongerKey",
			envValue: "1f0f29d72e880eec55360ea14bc18dfcbcc1a771dcd45fa03f0e5c181fdb368c664fb4329736f23566d4d5a2ba8af" +
				"98375a88a0907ba34bf715901942df2b580",
			expect: [32]byte{
				31, 15, 41, 215, 46, 136, 14, 236, 85, 54, 14, 161, 75, 193, 141, 252, 188, 193, 167, 113, 220, 212,
				95, 160, 63, 14, 92, 24, 31, 219, 54, 140,
			},
		},
		{
			name:     "ShorterKey",
			envValue: "087a92fbcde7afd24bab23ba428df42e1eb8d6197b677509",
			expect: [32]byte{
				8, 122, 146, 251, 205, 231, 175, 210, 75, 171, 35, 186, 66, 141, 244, 46, 30, 184, 214, 25, 123,
				103, 117, 9, 0, 0, 0, 0, 0, 0, 0, 0,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv(lib.MasterKeyEnv, testCase.envValue)

			ctx, err := lib.NewMasterKeyContext(t.Context())

			if testCase.expectErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)

			value, err := lib.MasterKeyContext(ctx)
			require.NoError(t, err)
			require.Equal(t, testCase.expect, value)
		})
	}
}
