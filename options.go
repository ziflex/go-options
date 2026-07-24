package options

import "errors"

type (
	Report func(ValidationError)

	Option[T any] func(*T, Report)
)

func Apply[T any](opts ...Option[T]) (T, error) {
	var zero T

	return applyInternal(zero, opts)
}

func ApplyWithValues[T any](values T, opts ...Option[T]) (T, error) {
	return applyInternal(values, opts)
}

// applyInternal applies the given options to the provided values and returns the modified values along with any errors encountered during the application of the options.
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
