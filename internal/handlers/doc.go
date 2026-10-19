// Package handlers holds the REST transport layer of the authentication service.
//
// Each handler decodes one request, calls a core operation and writes its result. A core
// sentinel error answers its own status code, and any other error answers 500. The handlers
// follow the OpenAPI specification at the repository root.
package handlers
