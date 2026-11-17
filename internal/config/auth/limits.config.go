package auth

import "github.com/a-novel-kit/golib/httpf"

// Limits caps what one caller, one mailbox or a burst can make the REST API spend. Rates are
// written "<requests>/<window>" in their variables.
type Limits struct {
	// Strict caps each caller of a route issuing anonymous tokens or invitation requests.
	Strict httpf.Rate
	// LoginBurst caps each caller's requests checking a password or short code, logins first, over a
	// short window. Each session's requests to the device routes count against it too, per session.
	LoginBurst httpf.Rate
	// Login caps the same requests over a longer window, per caller and per email or account named.
	Login httpf.Rate
	// Mailbox caps the emails each address receives from one route.
	Mailbox httpf.Rate
	// OAuthToken caps the token requests each client sends for each visitor.
	OAuthToken httpf.Rate
	// WaitlistMaxConcurrent caps the waitlist requests waiting on Google at once, per instance.
	WaitlistMaxConcurrent int
	// WaitlistMaxRequestSize caps a waitlist request body, in bytes.
	WaitlistMaxRequestSize int64
	// Argon2MaxConcurrent caps the password hashes running at once, per instance. It defaults to the
	// CPUs the container may use, since hashing is CPU-bound.
	Argon2MaxConcurrent int
}
