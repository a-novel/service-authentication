package auth

import (
	"errors"
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
func (cfg Waitlist) Validate() error {
	if cfg.URL == "" && cfg.Secret == "" {
		return nil
	}

	endpoint, err := url.Parse(cfg.URL)
	if err != nil {
		return ErrWaitlistConfig
	}

	switch {
	case endpoint.Scheme != "https", endpoint.Host != "script.google.com":
		return ErrWaitlistConfig
	case endpoint.User != nil, endpoint.RawQuery != "", endpoint.Fragment != "":
		return ErrWaitlistConfig
	case !strings.HasPrefix(endpoint.Path, "/macros/s/"), !strings.HasSuffix(endpoint.Path, "/exec"):
		return ErrWaitlistConfig
	case len(cfg.Secret) < waitlistMinSecretLength:
		return ErrWaitlistConfig
	case cfg.Timeout <= 0, cfg.Timeout > waitlistMaxTimeout:
		return ErrWaitlistConfig
	}

	return nil
}
