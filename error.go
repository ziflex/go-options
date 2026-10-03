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
	// OmitValue suppresses this error's value rendering and automatic fallback
	// value population. It does not redact child reasons, values, or field labels.
	OmitValue bool
}

// ToValidationError returns a ValidationError that describes the failure of a configuration value.
// A direct fieldless ValidationError or non-nil *ValidationError is copied and
// enriched with field and, unless OmitValue is true, a missing value. Other errors
// are preserved as the reason of a new wrapper. A direct error's OmitValue is
// propagated to that wrapper; wrapped and joined errors are not searched for it.
func ToValidationError(field, value string, err error) error {
	return normalizeValidationError(field, func() string { return value }, err)
}

func normalizeValidationError(field string, fallback func() string, err error) error {
	var omitValue bool

	switch validationErr := err.(type) {
	case ValidationError:
		omitValue = validationErr.OmitValue
		if validationErr.Field != "" {
			break
		}

		validationErr.Field = field
		if validationErr.Value == "" && !omitValue {
			validationErr.Value = fallback()
		}

		return validationErr
	case *ValidationError:
		if validationErr == nil {
			break
		}

		omitValue = validationErr.OmitValue
		if validationErr.Field != "" {
			break
		}

		normalized := *validationErr
		normalized.Field = field

		if normalized.Value == "" && !omitValue {
			normalized.Value = fallback()
		}

		return &normalized
	}

	normalized := ValidationError{
		Field:     field,
		Reason:    err,
		OmitValue: omitValue,
	}

	if !omitValue {
		normalized.Value = fallback()
	}

	return normalized
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

	if d.Value != "" && !d.OmitValue {
		b.WriteString(": value=")
		b.WriteString(d.Value)
	}

	return b.String()
}

// Unwrap returns the error that explains the validation failure.
func (d ValidationError) Unwrap() error {
	return d.Reason
}
