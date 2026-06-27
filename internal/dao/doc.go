// Package dao holds the Postgres data access layer of the authentication service.
//
// Each request type pairs with a single DAO struct exposing one Exec
// method, mirroring the core-layer convention. DAOs pull the active
// transaction off the context via postgres.GetContext, so callers must run them
// inside a postgres.RunInTx block (or otherwise install a transaction on the
// context) before calling Exec. The core consumes DAOs; nothing outside
// this package should issue SQL directly.
//
// SQL statements live in sibling .sql files embedded with go:embed; constraint
// violations are mapped to package-level sentinel errors (e.g.
// [ErrCredentialsInsertAlreadyExists]) so callers can branch on the
// integrity-failure case without parsing pgdriver errors themselves.
package dao
