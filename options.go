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

// ApplyTo applies each non-nil option to values in order. It returns the
// resulting value and any reported failures joined into one error.
func ApplyTo[T any](values T, opts ...Option[T]) (T, error) {
	return applyInternal(values, opts)
}

// ApplyWithValues applies opts to values in order.
//
// Deprecated: use ApplyTo instead.
func ApplyWithValues[T any](values T, opts ...Option[T]) (T, error) {
	return ApplyTo(values, opts...)
}

// New creates a reusable option constructor from setter and validators. Each
// constructed option delegates its behavior to With.
func New[C, V any](setter func(*C, V), validators ...Validator[V]) func(V) Option[C] {
	return func(value V) Option[C] {
		return With(value, setter, validators...)
	}
}

// With creates an option that validates value before passing it to setter.
// Every non-nil validator runs in order. The setter runs only when no validator
// reports a failure.
func With[C, V any](value V, setter func(*C, V), validators ...Validator[V]) Option[C] {
	return func(config *C, report Report) {
		valid := true
		validatorReport := func(err ValidationError) {
			valid = false
			report(err)
		}

		for _, validator := range validators {
			if validator == nil {
				continue
			}

			validator(value, validatorReport)
		}

		if valid {
			setter(config, value)
		}
	}
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
