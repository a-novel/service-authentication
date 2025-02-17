// Package dao (Data-Oriented Object) contains interfaces used to interact with a data source (database, filesystem,
// etc.)
//
// The interfaces used to interact with the data sources are called Repositories. Each repository describes a single
// action on a single data model, called Entity.
//
// Multiple DAO operations may be executed in a single transaction. A transaction will automatically revert all DAO
// operations within if a single of them fails. Such transactions may be initiated at higher levels of the application,
// such as in the services.
package dao
