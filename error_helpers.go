package options

import (
	"errors"
	"strings"
)

func normalizeValidationError(field string, err error) error {
	if err == nil {
		return nil
	}

	switch err := err.(type) {
	case ValidationError:
		if err.Field == "" {
			err.Field = field
		}

		return err
	case *ValidationError:
		if err == nil {
			return ValidationError{Field: field, Reason: "<nil>"}
		}
		if err.Field != "" {
			return err
		}

		normalized := *err
		normalized.Field = field

		return &normalized
	case interface {
		error
		Unwrap() []error
	}:
		if !containsValidationError(err) {
			return err
		}

		children := err.Unwrap()
		normalized := make([]error, len(children))

		for i, child := range children {
			normalized[i] = normalizeValidationError(field, child)
		}

		return errors.Join(normalized...)
	case interface {
		error
		Unwrap() error
	}:
		if !containsValidationError(err) {
			return err
		}

		child := err.Unwrap()
		normalized := normalizeValidationError(field, child)

		return wrappedValidationError{
			message: normalizedWrappedErrorMessage(err, child, normalized),
			err:     normalized,
		}
	default:
		return err
	}
}

func containsValidationError(err error) bool {
	var validationError ValidationError
	if errors.As(err, &validationError) {
		return true
	}

	var validationErrorPointer *ValidationError

	return errors.As(err, &validationErrorPointer)
}

type wrappedValidationError struct {
	message string
	err     error
}

func (e wrappedValidationError) Error() string {
	return e.message
}

func (e wrappedValidationError) Unwrap() error {
	return e.err
}

func normalizedWrappedErrorMessage(wrapper, child, normalized error) string {
	message := wrapper.Error()
	if validationError, ok := child.(*ValidationError); ok && validationError == nil {
		return message
	}

	childMessage := child.Error()
	if !strings.HasSuffix(message, childMessage) {
		return message
	}

	return strings.TrimSuffix(message, childMessage) + normalized.Error()
}
