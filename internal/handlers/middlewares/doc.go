// Package middlewares holds the REST middlewares of the authentication service. [Auth] verifies
// access tokens and checks permissions, then stores the verified claims on the request context
// for [session.GetClaimsContext]. [RestLimits] adds the strict rate limits of sensitive routes.
package middlewares
