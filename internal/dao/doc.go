// Package dao holds the Postgres data access layer of the authentication service.
//
// Each request type pairs with a single repository struct exposing one Exec
// method, mirroring the services-layer convention. Repositories pull the active
// transaction off the context via postgres.GetContext, so callers must run them
// inside a postgres.RunInTx block (or otherwise install a transaction on the
// context) before calling Exec. Services consume repositories; nothing outside
// this package should issue SQL directly.
//
// SQL statements live in sibling .sql files embedded with go:embed; constraint
// violations are mapped to package-level sentinel errors (e.g.
// [ErrCredentialsInsertAlreadyExists]) so callers can branch on the
// integrity-failure case without parsing pgdriver errors themselves.
package dao
