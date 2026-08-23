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
