package options

import "errors"

// Option is an alias for a function that configures a value of type T and
// returns any failure. Equivalent project-specific function types interoperate
// with Option without explicit conversions.
type Option[T any] = func(*T) error

// Apply creates the zero value of T and applies each non-nil option in order.
// It returns the resulting value and any failures joined into one error.
func Apply[T any](opts ...Option[T]) (T, error) {
	var zero T

	return applyInternal(zero, opts)
}

// ApplyTo applies each non-nil option to the initial value in order. It returns the
// resulting value and any failures joined into one error.
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

	for _, opt := range opts {
		if opt == nil {
			continue
		}

		if err := opt(&values); err != nil {
			errs = append(errs, err)
		}
	}

	return values, errors.Join(errs...)
}
