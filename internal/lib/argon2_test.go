package lib_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/lib"
)

func TestArgon2(t *testing.T) {
	t.Parallel()

	password := "password"

	encrypted, err := lib.GenerateArgon2(password, lib.Argon2ParamsDefault)
	require.NoError(t, err)
	require.NotEmpty(t, encrypted)

	testCases := []struct {
		name string

		password  string
		encrypted string

		expectErr error
	}{
		{
			name: "OK",

			password:  password,
			encrypted: encrypted,
		},
		{
			name: "WrongPassword",

			password:  "wrongpassword",
			encrypted: encrypted,

			expectErr: lib.ErrInvalidPassword,
		},
		{
			name: "Malformed/NotEnoughParts",

			password:  "password",
			encrypted: "malformed$",

			expectErr: lib.ErrInvalidHash,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err = lib.CompareArgon2(testCase.password, testCase.encrypted)
			require.ErrorIs(t, err, testCase.expectErr)
		})
	}
}
