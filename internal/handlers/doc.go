// Package handlers provides HTTP request handlers for the authentication API.
//
// Handlers translate HTTP requests to service calls and format responses.
// Error mapping converts service errors to appropriate HTTP status codes.
//
// Each handler follows the http.Handler interface pattern and includes
// OpenTelemetry tracing for observability.
package handlers
