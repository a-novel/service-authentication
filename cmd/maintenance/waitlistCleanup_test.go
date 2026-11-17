package main

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/config"
	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
)

func TestWaitlistCleanupOperation(t *testing.T) {
	t.Parallel()

	configured := config.App{Waitlist: authconfig.Waitlist{
		URL:     "https://script.google.com/macros/s/deployment/exec",
		Secret:  strings.Repeat("s", 32),
		Timeout: time.Second,
	}}

	testCases := []struct {
		name      string
		cfg       config.App
		args      []string
		apply     bool
		expectErr string
	}{
		{name: "Success/Preview", cfg: configured},
		{name: "Success/Apply", cfg: configured, args: []string{"--apply"}, apply: true},
		{name: "Success/ExplicitPreview", cfg: configured, args: []string{"--apply=false"}},
		{name: "Error/UnknownFlag", cfg: configured, args: []string{"--all"}, expectErr: "flag provided but not defined"},
		{
			name:      "Error/ExtraArgument",
			cfg:       configured,
			args:      []string{"anything"},
			expectErr: "unexpected maintenance operation parameters",
		},
		{name: "Error/WaitlistUnconfigured", expectErr: "configure the waitlist before cleanup: waitlist unavailable"},
		{
			name:      "Error/WaitlistInvalid",
			cfg:       config.App{Waitlist: authconfig.Waitlist{URL: configured.Waitlist.URL}},
			expectErr: "invalid waitlist configuration",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			operation, err := waitlistCleanupOperationDefinition().parse(testCase.args, testCase.cfg)
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
