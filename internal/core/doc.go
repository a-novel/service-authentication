// Package core holds the business-logic layer of the authentication service.
//
// Each core operation is a small struct constructed with its declared dependencies
// (its DAO interfaces, the json-keys signer, the SMTP sender, and other core
// operations it composes) and exposes a single Exec method that handles request
// validation, orchestration, and observability. Handlers consume the core; the
// core consumes the dao and lib packages.
//
// Sentinel errors such as [ErrInvalidRequest] signal expected user-facing outcomes and
// travel up to the handler, which maps them to a status code; infrastructure failures
// wrap the underlying error instead. Both are recorded on the operation's span through
// otel.ReportError.
package core
