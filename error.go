package options

import "strings"

// ValidationError describes a rejected configuration value.
type ValidationError struct {
	// Field identifies the name of the field that produced the error.
	Field string
	// Value identifies the invalid non-secret input.
	Value string
	// Reason explains why the configuration is invalid.
	Reason error
}

// ToValidationError returns a ValidationError that describes the failure of a configuration value.
// If err is already a ValidationError, it is returned with its Field and Value set if they are empty.
// Otherwise, a new ValidationError is returned with the provided field, value, and reason.
func ToValidationError(field, value string, err error) error {
	switch validationErr := err.(type) {
	case ValidationError:
		if validationErr.Field != "" {
			break
		}

		validationErr.Field = field
		if validationErr.Value == "" {
			validationErr.Value = value
		}

		return validationErr
	case *ValidationError:
		if validationErr == nil || validationErr.Field != "" {
			break
		}

		normalized := *validationErr
		normalized.Field = field
		if normalized.Value == "" {
			normalized.Value = value
		}

		return &normalized
	}

	return ValidationError{
		Field:  field,
		Value:  value,
		Reason: err,
	}
}

func (d ValidationError) Error() string {
	var b strings.Builder

	if d.Field != "" {
		b.WriteString(d.Field)
		b.WriteString(": ")
	}

	if d.Reason != nil {
		b.WriteString(d.Reason.Error())
	}

	if d.Value != "" {
		b.WriteString(": value=")
		b.WriteString(d.Value)
	}

	return b.String()
}

// Unwrap returns the error that explains the validation failure.
func (d ValidationError) Unwrap() error {
	return d.Reason
}
