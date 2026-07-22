// Package dao holds the Postgres data access layer of the authentication service.
//
// Each request type pairs with a single DAO struct exposing one Exec method, mirroring
// the core-layer convention. DAOs resolve their database handle from the context via
// postgres.GetContext, so they join whatever transaction the caller installed and commit
// statement by statement when there is none. A caller needing two writes to land together
// opens that scope in the core layer, through the injected transaction.Transactor. Nothing
// outside this package issues SQL directly.
//
// SQL statements live in sibling .sql files embedded with go:embed. Constraint violations
// map to package-level sentinel errors such as [ErrCredentialsInsertAlreadyExists], so
// callers branch on the integrity failure without parsing pgdriver errors themselves.
package dao
