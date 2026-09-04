package options

import (
	"errors"
	"fmt"
)

// Builder describes an option for configuration type C with value type V.
// Create builders with New.
type Builder[C, V any] struct {
	setter           func(*C, V)
	value            V
	hasValue         bool
	defaultValue     V
	defaultPredicate Predicate[V]
	hasDefault       bool
	name             string
	validators       []Validator[V]
}

// New creates a builder that uses setter to apply its value.
func New[C, V any](setter func(*C, V)) Builder[C, V] {
	return Builder[C, V]{setter: setter}
}

// Value binds value to the option. Repeated calls replace the previous value.
func (b Builder[C, V]) Value(value V) Builder[C, V] {
	b.value = value
	b.hasValue = true

	return b
}

// Default binds a fallback used when Value was not called or its bound value
// is the zero value of its dynamic type. Repeated calls to Default or
// DefaultWhen replace the previous fallback policy. Default and Value may be
// called in either order.
func (b Builder[C, V]) Default(defaultValue V) Builder[C, V] {
	b.defaultValue = defaultValue
	b.defaultPredicate = isZero[V]
	b.hasDefault = true

	return b
}

// DefaultWhen binds a fallback used when Value was not called or predicate
// matches its bound value. A nil predicate never matches a bound value.
// Repeated calls to Default or DefaultWhen replace the previous fallback
// policy. DefaultWhen and Value may be called in either order.
func (b Builder[C, V]) DefaultWhen(defaultValue V, predicate Predicate[V]) Builder[C, V] {
	b.defaultValue = defaultValue
	b.defaultPredicate = predicate
	b.hasDefault = true

	return b
}

// Named sets the option name used when Build describes validation failures.
// Repeated calls replace the previous name.
func (b Builder[C, V]) Named(name string) Builder[C, V] {
	b.name = name

	return b
}

// Validators appends validators to the option in execution order.
func (b Builder[C, V]) Validators(validators ...Validator[V]) Builder[C, V] {
	combined := make([]Validator[V], 0, len(b.validators)+len(validators))
	combined = append(combined, b.validators...)
	combined = append(combined, validators...)
	b.validators = combined

	return b
}

// Build produces the configured option. If neither Value nor a default was
// configured, the option returns a validation failure when applied.
func (b Builder[C, V]) Build() Option[C] {
	setter := b.setter
	value := b.value
	hasValue := b.hasValue

	if b.hasDefault && (!hasValue || (b.defaultPredicate != nil && b.defaultPredicate(value))) {
		value = b.defaultValue
		hasValue = true
	}

	name := b.name
	validators := append([]Validator[V](nil), b.validators...)

	return func(config *C) error {
		if !hasValue {
			return ValidationError{Reason: errors.New("option value was not provided")}
		}

		var errs []error

		for _, validator := range validators {
			if validator == nil {
				continue
			}

			if err := validator(value); err != nil {
				errs = append(errs, ToValidationError(name, fmt.Sprint(value), err))
			}
		}

		if err := errors.Join(errs...); err != nil {
			return err
		}

		setter(config, value)

		return nil
	}
}
