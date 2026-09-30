package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWaitlistCleanupOperation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		args      []string
		apply     bool
		expectErr string
	}{
		{name: "PreviewByDefault"},
		{name: "Apply", args: []string{"--apply"}, apply: true},
		{name: "ExplicitPreview", args: []string{"--apply=false"}},
		{name: "RejectUnknownFlag", args: []string{"--all"}, expectErr: "flag provided but not defined"},
		{name: "RejectExtraArgument", args: []string{"anything"}, expectErr: "unexpected maintenance operation parameters"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			operation, err := waitlistCleanupOperationDefinition().parse(testCase.args)
			if testCase.expectErr != "" {
				require.ErrorContains(t, err, testCase.expectErr)
				require.Nil(t, operation)

				return
			}

			require.NoError(t, err)
			require.Equal(t, &waitlistCleanupOperation{apply: testCase.apply}, operation)
		})
	}
}
