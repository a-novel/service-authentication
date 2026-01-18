// Package dao provides data access objects for PostgreSQL database operations.
//
// This package implements the repository pattern for credentials and short codes,
// providing type-safe database operations with proper error handling and OTEL tracing.
//
// All operations use parameterized queries to prevent SQL injection.
package dao
