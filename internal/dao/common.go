package dao

import (
	"errors"

	"github.com/uptrace/bun/driver/pgdriver"
)

// pgUniqueViolation is the SQLSTATE PostgreSQL raises when a write breaks a unique index.
const pgUniqueViolation = "23505"

// isUniqueViolation reports whether err comes from a write that broke a unique index.
func isUniqueViolation(err error) bool {
	var pgErr pgdriver.Error

	return errors.As(err, &pgErr) && pgErr.Field('C') == pgUniqueViolation
}
