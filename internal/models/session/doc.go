// Package session defines the JWT payloads of a session's tokens: the access token a client sends
// with each request, and the refresh token that renews it. JSON Keys signs and verifies both.
//
// The core layer issues the tokens, and the auth middleware and the Go client decode the access
// token. The payloads stay outside the core layer, so the Go client links them without the service.
package session
