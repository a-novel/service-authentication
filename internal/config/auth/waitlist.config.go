package auth

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// ErrWaitlistConfig is returned when a partially configured or unsafe writer is supplied.
var ErrWaitlistConfig = errors.New("invalid waitlist configuration")

const (
	// waitlistMinSecretLength rejects signing keys shorter than 32 bytes.
	waitlistMinSecretLength = 32
	// waitlistMaxTimeout bounds the optional Google dependency on the request path.
	waitlistMaxTimeout = 20 * time.Second
	// WaitlistBatchSize bounds private maintenance reads and removals; the Apps Script uses the same limit.
	WaitlistBatchSize = 100
	// WaitlistMaxEntries caps the temporary sheet and the work performed by one cleanup run.
	WaitlistMaxEntries = 10000
)

// Waitlist configures the private Google Apps Script invitation-list writer.
// Leaving both URL and Secret empty disables the integration.
type Waitlist struct {
	// URL is the HTTPS /exec deployment URL on script.google.com.
	URL string `json:"url" yaml:"url"`
	// Secret authenticates requests and is excluded from configuration serialization.
	Secret string `json:"-" yaml:"-"`
	// Timeout bounds each Google request, including its response redirect.
	Timeout time.Duration `json:"timeout" yaml:"timeout"`
}

// Validate rejects partial configuration, non-Google endpoints, and unbounded requests.
// Failures wrap [ErrWaitlistConfig] with the required setting, without exposing configuration values.
func (cfg Waitlist) Validate() error {
	if cfg.URL == "" && cfg.Secret == "" {
		return nil
	}

	if cfg.URL == "" || cfg.Secret == "" {
		return fmt.Errorf("waitlist URL and secret must both be set or both be empty: %w", ErrWaitlistConfig)
	}

	endpoint, err := url.Parse(cfg.URL)
	if err != nil {
		// Parse errors include the raw URL, which may contain accidentally embedded credentials.
		return fmt.Errorf("waitlist URL cannot be parsed: %w", ErrWaitlistConfig)
	}

	switch {
	case endpoint.Scheme != "https", endpoint.Host != "script.google.com":
		return fmt.Errorf("waitlist URL must use https://script.google.com: %w", ErrWaitlistConfig)
	case endpoint.User != nil, endpoint.RawQuery != "", endpoint.Fragment != "":
		return fmt.Errorf("waitlist URL must not contain credentials, a query, or a fragment: %w", ErrWaitlistConfig)
	case !strings.HasPrefix(endpoint.Path, "/macros/s/"), !strings.HasSuffix(endpoint.Path, "/exec"):
		return fmt.Errorf("waitlist URL must use /macros/s/<deployment>/exec: %w", ErrWaitlistConfig)
	case len(cfg.Secret) < waitlistMinSecretLength:
		return fmt.Errorf("waitlist secret must contain at least %d bytes: %w", waitlistMinSecretLength, ErrWaitlistConfig)
	case cfg.Timeout <= 0, cfg.Timeout > waitlistMaxTimeout:
		return fmt.Errorf("waitlist timeout must be greater than zero and at most %s: %w",
			waitlistMaxTimeout, ErrWaitlistConfig)
	}

	return nil
}
