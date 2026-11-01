package handlers

import (
	"errors"
	"mime"
	"net/http"
	"net/url"

	"go.opentelemetry.io/otel/trace"

	"github.com/a-novel-kit/golib/httpf"

	"github.com/a-novel/service-authentication/v2/internal/core"
)

// formMediaType is the media type of an OAuth 2.0 request body (RFC 6749 Appendix B).
const formMediaType = "application/x-www-form-urlencoded"

// RestOAuthError is the JSON body of an OAuth 2.0 error response (RFC 6749 §4.1.2.1, §5.2).
//
//nolint:tagliatelle // RFC 6749 names the members.
type RestOAuthError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
	// RedirectURI is where the account platform sends an authorization error response, with the
	// request's state and its own iss. It is absent when the account platform must not redirect.
	RedirectURI string `json:"redirect_uri,omitempty"`
}

// sendOAuthError answers err with its OAuth 2.0 error response, 400, when err is an
// [core.OAuthError], and reports whether it did.
func sendOAuthError(w http.ResponseWriter, span trace.Span, err error) bool {
	var refusal *core.OAuthError
	if !errors.As(err, &refusal) {
		return false
	}

	setNoStore(w)
	httpf.SendJSONStatus(w, span, http.StatusBadRequest, RestOAuthError{
		Error:            refusal.Code,
		ErrorDescription: refusal.Description,
		RedirectURI:      refusal.RedirectURI,
	})

	return true
}

// setNoStore keeps a response that carries a code, a token or a refusal of one out of every cache
// (RFC 6749 §5.1).
func setNoStore(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
}

// readOAuthForm returns the parameters of an application/x-www-form-urlencoded request body, or
// [core.ErrOAuthMalformedRequest] for a body of another type or one that does not parse.
func readOAuthForm(r *http.Request) (url.Values, error) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != formMediaType {
		return nil, core.ErrOAuthMalformedRequest
	}

	err = r.ParseForm()
	if err != nil {
		return nil, errors.Join(err, core.ErrOAuthMalformedRequest)
	}

	return r.PostForm, nil
}
