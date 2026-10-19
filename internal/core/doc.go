// Package core holds the business logic of the authentication service.
//
// Each operation validates its request, applies the account or session rules, and owns the
// transaction that makes its writes atomic. Handlers call core operations, which reach the dao
// and lib packages through interfaces they declare. A sentinel error such as [ErrInvalidRequest]
// marks an outcome the caller caused, and handlers map it to a status code.
package core
