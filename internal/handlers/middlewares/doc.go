// Package middlewares holds the REST middlewares of the authentication service. [Auth] verifies
// access tokens and checks permissions, then stores the verified claims on the request context
// for [session.GetClaimsContext], and [RestRefuseClientTokens] keeps an OAuth client's token off the
// routes that manage accounts. [RestLimits] adds the strict rate limits of sensitive routes.
//
// The middleware that refuses signed-out sessions, handlers.RestRefuseEndedSessions, calls the core,
// so it sits in the handlers package: the Go client links this package and must not link the core.
package middlewares
