// Package middlewares holds the REST middlewares of the authentication service. [Auth] verifies
// access tokens and checks permissions, then stores the verified claims on the request context
// for [session.GetClaimsContext], and [RestRefuseClientTokens] keeps an OAuth client's token off the
// routes that manage accounts. [RestLimits] adds the strict rate limits of sensitive routes.
// [RestServiceToken] admits only the services allowed to call, before the router reads anything else.
package middlewares
