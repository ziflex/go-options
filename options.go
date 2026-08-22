package options

import "errors"

type (
	// Report records a validation failure encountered while applying an option.
	Report func(ValidationError)

	// Option configures a value of type T and reports any validation failures.
	Option[T any] func(*T, Report)
)

// Apply creates the zero value of T and applies each non-nil option in order.
// It returns the resulting value and any reported failures joined into one error.
func Apply[T any](opts ...Option[T]) (T, error) {
	var zero T

	return applyInternal(zero, opts)
}

// ApplyTo applies each non-nil option to the initial value in order. It returns the
// resulting value and any reported failures joined into one error.
func ApplyTo[T any](initial T, opts ...Option[T]) (T, error) {
	return applyInternal(initial, opts)
}

// ApplyWithValues applies opts to values in order.
//
// Deprecated: use ApplyTo instead.
func ApplyWithValues[T any](values T, opts ...Option[T]) (T, error) {
	return ApplyTo(values, opts...)
}

func applyInternal[T any](values T, opts []Option[T]) (T, error) {
	var errs []error

	reporter := func(e ValidationError) {
		errs = append(errs, e)
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}

		opt(&values, reporter)
	}

	if len(errs) > 0 {
		return values, errors.Join(errs...)
	}

	return values, nil
}
