// Package dao holds the Postgres data access layer of the authentication service.
//
// Each request type pairs with a single DAO struct exposing one Exec
// method, mirroring the core-layer convention. DAOs resolve their database
// handle from the context via postgres.GetContext, and work either way: with no
// transaction installed each statement commits on its own, and when a caller has
// opened one they take part in it without knowing. Callers that need two writes
// to land together open that scope in the core layer, through the injected
// transaction.Transactor. The core consumes DAOs; nothing outside this package
// should issue SQL directly.
//
// SQL statements live in sibling .sql files embedded with go:embed; constraint
// violations are mapped to package-level sentinel errors (e.g.
// [ErrCredentialsInsertAlreadyExists]) so callers can branch on the
// integrity-failure case without parsing pgdriver errors themselves.
package dao
