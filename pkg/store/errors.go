package store

import (
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"modernc.org/sqlite"
)

func MapError(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	if errors.IsNotFound(err) {
		return errors.NotFoundf(format, args...)
	}
	if errors.IsConflict(err) || isUniqueConstraint(err) {
		return errors.Conflictf(format, args...)
	}
	if errors.IsInvalid(err) {
		return errors.Invalidf(format, args...)
	}
	return errors.Internalf(format+": %w", append(args, err)...)
}

func isUniqueConstraint(err error) bool {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}
	// SQLite extended result codes for PRIMARY KEY and UNIQUE constraints.
	return sqliteErr.Code() == 1555 || sqliteErr.Code() == 2067
}

func isBusy(err error) bool {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}
	primaryCode := sqliteErr.Code() & 0xff
	return primaryCode == 5 || primaryCode == 6
}
