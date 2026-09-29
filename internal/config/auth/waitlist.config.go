package auth

import (
	"errors"
	"net/url"
	"strings"
	"time"
)

// ErrWaitlistConfig is returned when a partially configured or unsafe writer is supplied.
var ErrWaitlistConfig = errors.New("invalid waitlist configuration")

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
	if err != nil || endpoint.Scheme != "https" || endpoint.Host != "script.google.com" ||
		endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" ||
		!strings.HasPrefix(endpoint.Path, "/macros/s/") || !strings.HasSuffix(endpoint.Path, "/exec") ||
		len(cfg.Secret) < 32 || cfg.Timeout <= 0 || cfg.Timeout > 20*time.Second {
		return ErrWaitlistConfig
	}

	return nil
}
