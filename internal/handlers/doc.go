// Package handlers holds the REST transport layer of the authentication service.
//
// Each handler wraps a single service: it decodes the HTTP request, calls
// service.Exec, and translates the result (or error) into a status code and
// body. Service-layer sentinel errors map to user-facing 4xx responses; any
// other error surfaces as 500. The OpenAPI specification at the repository
// root is the source of truth for request and response shapes — handlers
// mirror it.
package handlers
