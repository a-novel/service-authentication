package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"slices"
	"strings"
)

// ErrClientsConfig is returned by [ParseClients] for a client registry that breaks a rule of
// [ParseClients].
var ErrClientsConfig = errors.New("invalid client registry")

// Client is a platform that signs people in through the account platform, with the OAuth 2.0
// authorization code grant (RFC 6749 §4.1) and PKCE (RFC 7636). It is a confidential client: it
// authenticates with its Kubernetes service token (RFC 6749 §2.3.2).
type Client struct {
	// RedirectURI is the client's one redirection endpoint. An authorization request names it
	// exactly, or names none.
	RedirectURI string `json:"redirectURI" yaml:"redirectURI"`
	// Caller is the client's Kubernetes caller, as its service token names it, such as
	// system:serviceaccount:studio:web. A request with this caller authenticates as the client.
	Caller string `json:"caller" yaml:"caller"`
}

// OAuth configures who takes part in the authorization code grant.
type OAuth struct {
	// AccountPlatformCaller is the Kubernetes caller of the account platform, the only caller that
	// validates authorization requests and issues codes.
	AccountPlatformCaller string `json:"accountPlatformCaller" yaml:"accountPlatformCaller"`
	// Clients is the client registry, keyed by client ID, such as studio.
	Clients map[string]Client `json:"clients" yaml:"clients"`
}

// Check refuses a registry whose client shares the account platform's caller, which would let one
// workload both issue and exchange codes. When serviceTokenCallers is not nil, the server verifies
// service tokens and admits those callers alone, so Check also refuses a registry whose callers, or
// the account platform's, it lacks: the server would refuse them before the grant sees them.
func (oauth OAuth) Check(serviceTokenCallers []string) error {
	for clientID, client := range oauth.Clients {
		if client.Caller == oauth.AccountPlatformCaller {
			return fmt.Errorf("%w: client %q has the account platform's caller", ErrClientsConfig, clientID)
		}

		if serviceTokenCallers != nil && !slices.Contains(serviceTokenCallers, client.Caller) {
			return fmt.Errorf("%w: the service tokens do not admit client %q's caller", ErrClientsConfig, clientID)
		}
	}

	if len(oauth.Clients) > 0 && serviceTokenCallers != nil &&
		!slices.Contains(serviceTokenCallers, oauth.AccountPlatformCaller) {
		return fmt.Errorf("%w: the service tokens do not admit the account platform's caller", ErrClientsConfig)
	}

	return nil
}

// ParseClients reads a client registry from its JSON form, an object keyed by client ID:
//
//	{
//	  "studio": {
//	    "redirectURI": "https://studio.storyverse.fr/auth/callback",
//	    "caller": "system:serviceaccount:studio:web"
//	  }
//	}
//
// It refuses a registry that breaks a rule a client must follow:
//   - The client ID is printable ASCII (RFC 6749 §2.2, Appendix A.1).
//   - The redirect URI is absolute and has no fragment (RFC 6749 §3.1.2).
//   - The redirect URI uses https, or http on a loopback host for a local stack (RFC 9700 §2.6).
//   - Each client has a caller of its own, since a token request finds its client by the caller.
func ParseClients(value string) (map[string]Client, error) {
	var clients map[string]Client

	err := json.Unmarshal([]byte(value), &clients)
	if err != nil {
		return nil, errors.Join(err, ErrClientsConfig)
	}

	callers := make(map[string]string, len(clients))

	for clientID, client := range clients {
		defect := clientDefect(clientID, client)
		if defect != "" {
			return nil, fmt.Errorf("%w: client %q: %s", ErrClientsConfig, clientID, defect)
		}

		if other, taken := callers[client.Caller]; taken {
			return nil, fmt.Errorf("%w: clients %q and %q share a caller", ErrClientsConfig, other, clientID)
		}

		callers[client.Caller] = clientID
	}

	return clients, nil
}

// clientDefect returns the rule of [ParseClients] that a client breaks on its own, or an empty string.
func clientDefect(clientID string, client Client) string {
	if clientID == "" || strings.ContainsFunc(clientID, func(r rune) bool { return r < 0x20 || r > 0x7e }) {
		return "the client ID is not printable ASCII"
	}

	if client.Caller == "" {
		return "no caller"
	}

	redirectURI, err := url.Parse(client.RedirectURI)

	switch {
	case err != nil || !redirectURI.IsAbs() || redirectURI.Host == "":
		return "the redirect URI is not absolute"
	case strings.Contains(client.RedirectURI, "#"):
		return "the redirect URI has a fragment"
	case redirectURI.Scheme == "https":
		return ""
	case redirectURI.Scheme == "http" && isLoopback(redirectURI.Hostname()):
		return ""
	}

	return "the redirect URI uses neither https nor http on a loopback host"
}

// isLoopback reports whether host names the local machine.
func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}

	ip := net.ParseIP(host)

	return ip != nil && ip.IsLoopback()
}
