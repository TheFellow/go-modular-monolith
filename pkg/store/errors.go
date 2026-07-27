package store

import (
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/mjl-/bstore"
)

// MapError converts bstore errors to domain errors.
// Use in DAO methods to ensure consistent error handling.
func MapError(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, bstore.ErrAbsent) {
		return errors.NotFoundf(format, args...)
	}
	if errors.Is(err, bstore.ErrUnique) {
		return errors.Conflictf(format, args...)
	}
	if errors.Is(err, bstore.ErrZero) {
		return errors.Invalidf(format, args...)
	}
	return errors.Internalf(format+": %w", append(args, err)...)
}
