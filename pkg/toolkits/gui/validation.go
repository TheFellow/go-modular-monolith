package gui

import "github.com/TheFellow/go-modular-monolith/pkg/errors"

// Validator is a composable, presentation-only validation function.
type Validator func(string) error

// Validate applies validators in order and returns the first failure.
func Validate(value string, validators ...Validator) error {
	for _, validate := range validators {
		if validate == nil {
			continue
		}
		if err := validate(value); err != nil {
			return err
		}
	}
	return nil
}

// Required creates a validator while allowing each bespoke surface to supply
// its own field-specific message.
func Required(message string) Validator {
	return func(value string) error {
		if value == "" {
			return errors.New(message)
		}
		return nil
	}
}
