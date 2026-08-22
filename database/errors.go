package database

import (
	"strings"

	"gorm.io/gorm"
)

// isDuplicateKeyError reports whether the error is a SQLite unique-constraint
// violation (e.g. a duplicate primary key on a relationship table). GORM does
// not expose this cleanly, so we match against the known SQLite error text.
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	if err == gorm.ErrDuplicatedKey {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "constraint failed") ||
		strings.Contains(msg, "duplicate")
}
