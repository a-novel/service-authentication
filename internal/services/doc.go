// Package services holds the business-logic layer of the authentication service.
//
// Each service is a small struct constructed with its declared dependencies (DAO
// repositories, the json-keys signer, the SMTP sender, and other services it
// composes) and exposes a single Exec method that handles request validation,
// orchestration, and observability. Handlers consume services; services consume
// the dao and lib packages.
//
// Sentinel errors (e.g. [ErrInvalidRequest]) signal expected user-facing outcomes
// and are returned directly without marking the surrounding span as failed.
// Genuine infrastructure failures wrap the underlying error and are reported on
// the span via otel.ReportError.
package services
