package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuthorizationCodePurgeOperation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		args      []string
		expectErr string
	}{
		{name: "Success"},
		{name: "Error/UnknownFlag", args: []string{"--all"}, expectErr: "flag provided but not defined"},
		{name: "Error/ExtraArgument", args: []string{"anything"}, expectErr: "unexpected maintenance operation parameters"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			operation, err := authorizationCodePurgeOperationDefinition().parse(testCase.args)
			if testCase.expectErr != "" {
				require.ErrorContains(t, err, testCase.expectErr)
				require.Nil(t, operation)

				return
			}

			require.NoError(t, err)
			require.Equal(t, &authorizationCodePurgeOperation{}, operation)
		})
	}
}
