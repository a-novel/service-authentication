// Package session defines the JWT payloads of a session's tokens: the access token a client sends
// with each request, and the refresh token that renews it. JSON Keys signs and verifies both.
//
// The core layer issues the tokens, and the auth middleware and the Go client decode the access
// token. The middleware then carries the verified claims to the handlers on the request context.
// The package stays outside the service layers, so the Go client links it without the service.
package session
