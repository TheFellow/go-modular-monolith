package forms

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Validator validates a field value.
type Validator func(value any) error

// Required returns a validator that rejects empty values.
func Required() Validator {
	return func(value any) error {
		if isEmptyValue(value) {
			return errors.New("required")
		}
		return nil
	}
}

// MinLength validates a minimum string length.
func MinLength(n int) Validator {
	return func(value any) error {
		str, ok := value.(string)
		if !ok {
			return nil
		}
		if utf8.RuneCountInString(str) < n {
			return fmt.Errorf("minimum length is %d", n)
		}
		return nil
	}
}

// MaxLength validates a maximum string length.
func MaxLength(n int) Validator {
	return func(value any) error {
		str, ok := value.(string)
		if !ok {
			return nil
		}
		if utf8.RuneCountInString(str) > n {
			return fmt.Errorf("maximum length is %d", n)
		}
		return nil
	}
}

// Pattern validates a string using a regex.
func Pattern(regex string) Validator {
	re := regexp.MustCompile(regex)
	return func(value any) error {
		str, ok := value.(string)
		if !ok || strings.TrimSpace(str) == "" {
			return nil
		}
		if !re.MatchString(str) {
			return errors.New("invalid format")
		}
		return nil
	}
}

// Min validates a minimum numeric value.
func Min(n float64) Validator {
	return func(value any) error {
		val, ok, err := numericValue(value)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		if val < n {
			return fmt.Errorf("must be at least %g", n)
		}
		return nil
	}
}

// Max validates a maximum numeric value.
func Max(n float64) Validator {
	return func(value any) error {
		val, ok, err := numericValue(value)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		if val > n {
			return fmt.Errorf("must be at most %g", n)
		}
		return nil
	}
}

func isEmptyValue(value any) bool {
	if value == nil {
		return true
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed) == ""
	case []byte:
		return strings.TrimSpace(string(typed)) == ""
	case fmt.Stringer:
		return strings.TrimSpace(typed.String()) == ""
	default:
		return false
	}
}

func numericValue(value any) (float64, bool, error) {
	if value == nil {
		return 0, false, nil
	}
	switch typed := value.(type) {
	case float64:
		return typed, true, nil
	case float32:
		return float64(typed), true, nil
	case int:
		return float64(typed), true, nil
	case int64:
		return float64(typed), true, nil
	case int32:
		return float64(typed), true, nil
	case int16:
		return float64(typed), true, nil
	case int8:
		return float64(typed), true, nil
	case uint:
		return float64(typed), true, nil
	case uint64:
		return float64(typed), true, nil
	case uint32:
		return float64(typed), true, nil
	case uint16:
		return float64(typed), true, nil
	case uint8:
		return float64(typed), true, nil
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return 0, false, nil
		}
		val, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return 0, true, errors.New("invalid number")
		}
		return val, true, nil
	default:
		return 0, false, nil
	}
}
