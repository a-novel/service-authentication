// Package dao holds the data access layer of the authentication service: the Postgres queries,
// and the client of the Google Sheet that stores the invitation list.
//
// A Postgres DAO resolves its database handle from the context with postgres.GetContext, so it
// joins the transaction its caller opened and commits statement by statement otherwise. A core
// operation opens that transaction when two writes must land together. [ShortCodeInsert] alone
// runs a transaction of its own.
//
// SQL lives in sibling .sql files embedded with go:embed. A database outcome a caller branches on,
// such as a missing row or a unique violation, is joined onto the driver error as a package
// sentinel, so callers match it with errors.Is.
package dao
