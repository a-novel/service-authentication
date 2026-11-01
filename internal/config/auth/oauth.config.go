package auth

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ErrClientsConfig is returned by [ParseClients] for a client registry that names a client without
// its redirect URI or caller, or two clients with one caller.
var ErrClientsConfig = errors.New("invalid client registry")

// Client is a platform that signs people in through the account platform, with the OAuth 2.0
// authorization code grant (RFC 6749 §4.1) and PKCE (RFC 7636).
type Client struct {
	// RedirectURI is the only URI the account platform sends a code to. A request names it exactly,
	// or gets no code.
	RedirectURI string `json:"redirectURI" yaml:"redirectURI"`
	// Caller is the client's Kubernetes caller, as its service token names it, such as
	// system:serviceaccount:studio:web. Only this caller exchanges the client's codes.
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

// ParseClients reads a client registry from its JSON form, an object keyed by client ID:
//
//	{
//	  "studio": {
//	    "redirectURI": "https://studio.storyverse.fr/auth/callback",
//	    "caller": "system:serviceaccount:studio:web"
//	  }
//	}
//
// Each client needs both members, and its own caller, since a code exchange finds its client by
// the caller.
func ParseClients(value string) (map[string]Client, error) {
	var clients map[string]Client

	err := json.Unmarshal([]byte(value), &clients)
	if err != nil {
		return nil, errors.Join(err, ErrClientsConfig)
	}

	callers := make(map[string]string, len(clients))

	for clientID, client := range clients {
		if client.RedirectURI == "" || client.Caller == "" {
			return nil, fmt.Errorf("%w: client %q needs a redirect URI and a caller", ErrClientsConfig, clientID)
		}

		if other, taken := callers[client.Caller]; taken {
			return nil, fmt.Errorf("%w: clients %q and %q share a caller", ErrClientsConfig, other, clientID)
		}

		callers[client.Caller] = clientID
	}

	return clients, nil
}
