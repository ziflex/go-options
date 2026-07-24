package options

import "strings"

// ValidationError represents an error that occurs when validating configuration options. It contains information about the field that caused the error, the invalid value, and the reason for the validation failure.
type ValidationError struct {
	// Field identifies the name of the field that produced the error.
	Field string
	// Value identifies the invalid non-secret input.
	Value string
	// Reason explains why the configuration is invalid.
	Reason string
}

func (d ValidationError) Error() string {
	var b strings.Builder

	if d.Field != "" {
		b.WriteString(d.Field)
		b.WriteString(": ")
	}

	b.WriteString(d.Reason)

	if d.Value != "" {
		b.WriteString(": value=")
		b.WriteString(d.Value)
	}

	return b.String()
}
