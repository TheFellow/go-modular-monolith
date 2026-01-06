package quality

import "github.com/TheFellow/go-modular-monolith/pkg/errors"

type Quality string

const (
	Equivalent Quality = "equivalent"
	Similar    Quality = "similar"
	Different  Quality = "different"
)

func (q Quality) Rank() int {
	switch q {
	case Equivalent:
		return 3
	case Similar:
		return 2
	case Different:
		return 1
	default:
		return 0
	}
}

func (q Quality) Validate() error {
	switch q {
	case Equivalent, Similar, Different:
		return nil
	default:
		return errors.Invalidf("invalid quality %q", string(q))
	}
}
