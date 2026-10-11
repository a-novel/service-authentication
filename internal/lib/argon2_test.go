package lib_test

import (
	"testing"
	"time"

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

//nolint:paralleltest // It resizes the process-wide hash gate, which parallel tests share.
func TestLimitArgon2(t *testing.T) {
	lib.LimitArgon2(1)
	t.Cleanup(func() { lib.LimitArgon2(4) })

	// Every hash must give its slot back, or the next one waits forever.
	hashes := make(chan error, 3)
	for range cap(hashes) {
		go func() {
			_, err := lib.GenerateArgon2("password", lib.Argon2ParamsDefault)
			hashes <- err
		}()
	}

	for range cap(hashes) {
		select {
		case err := <-hashes:
			require.NoError(t, err)
		case <-time.After(30 * time.Second):
			require.FailNow(t, "a hash never got a slot")
		}
	}
}
