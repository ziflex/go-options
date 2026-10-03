package options

import (
	"cmp"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type (
	// Validator validates a value. It returns nil when the value is valid and an
	// error describing why validation failed otherwise. Validators should treat the
	// value as read-only. Use errors.Join to return multiple failures.
	Validator[V any] func(V) error

	orderedNumber interface {
		~int | ~int8 | ~int16 | ~int32 | ~int64 |
			~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
			~float32 | ~float64
	}
)

// Check adapts check into a Validator while preserving its returned error.
func Check[V any](check func(V) error) Validator[V] {
	return check
}

// NotNil rejects nil values, including typed nil values stored in interfaces.
// Values whose type cannot be nil always pass validation.
func NotNil[V any]() Validator[V] {
	return func(value V) error {
		reflected := reflect.ValueOf(value)
		if !reflected.IsValid() {
			return ValidationError{
				Value:  "<nil>",
				Reason: errors.New("must not be nil"),
			}
		}

		switch reflected.Kind() {
		case reflect.Chan,
			reflect.Func,
			reflect.Interface,
			reflect.Map,
			reflect.Pointer,
			reflect.Slice,
			reflect.UnsafePointer:
			if reflected.IsNil() {
				return ValidationError{
					Value:  "<nil>",
					Reason: errors.New("must not be nil"),
				}
			}
		}

		return nil
	}
}

// NotNilPtr rejects nil pointers without using reflection. Use NotNil when the
// value may be another nil-capable type.
func NotNilPtr[V any]() Validator[*V] {
	return func(value *V) error {
		if value == nil {
			return ValidationError{Reason: errors.New("cannot be nil")}
		}

		return nil
	}
}

// NotZero rejects the zero value of V.
func NotZero[V comparable]() Validator[V] {
	return func(value V) error {
		var zero V

		if value == zero {
			return ValidationError{
				Value:  fmt.Sprint(value),
				Reason: errors.New("must not be zero"),
			}
		}

		return nil
	}
}

// Positive rejects values that are not greater than zero.
func Positive[V orderedNumber]() Validator[V] {
	return func(value V) error {
		var zero V

		if !(value > zero) {
			return ValidationError{
				Value:  fmt.Sprint(value),
				Reason: errors.New("must be positive"),
			}
		}

		return nil
	}
}

// NonNegative rejects values that are not greater than or equal to zero.
func NonNegative[V orderedNumber]() Validator[V] {
	return func(value V) error {
		var zero V

		if !(value >= zero) {
			return ValidationError{
				Value:  fmt.Sprint(value),
				Reason: errors.New("must be non-negative"),
			}
		}

		return nil
	}
}

// Negative rejects values that are not less than zero.
func Negative[V orderedNumber]() Validator[V] {
	return func(value V) error {
		var zero V

		if !(value < zero) {
			return ValidationError{
				Value:  fmt.Sprint(value),
				Reason: errors.New("must be negative"),
			}
		}

		return nil
	}
}

// NonPositive rejects values that are not less than or equal to zero.
func NonPositive[V orderedNumber]() Validator[V] {
	return func(value V) error {
		var zero V

		if !(value <= zero) {
			return ValidationError{
				Value:  fmt.Sprint(value),
				Reason: errors.New("must be non-positive"),
			}
		}

		return nil
	}
}

// NotEmpty rejects empty string values.
func NotEmpty[S ~string]() Validator[S] {
	return func(value S) error {
		if value == "" {
			return ValidationError{
				Value:  strconv.Quote(string(value)),
				Reason: errors.New("must not be empty"),
			}
		}

		return nil
	}
}

// NotBlank rejects empty strings and strings containing only Unicode whitespace.
func NotBlank[S ~string]() Validator[S] {
	return func(value S) error {
		if strings.TrimSpace(string(value)) == "" {
			return ValidationError{
				Value:  strconv.Quote(string(value)),
				Reason: errors.New("must not be blank"),
			}
		}

		return nil
	}
}

// Min rejects values that are not greater than or equal to minimum.
func Min[V cmp.Ordered](minimum V) Validator[V] {
	return func(value V) error {
		if !(value >= minimum) {
			return ValidationError{
				Value:  fmt.Sprint(value),
				Reason: fmt.Errorf("must be greater than or equal to %v", minimum),
			}
		}

		return nil
	}
}

// Max rejects values that are not less than or equal to maximum.
func Max[V cmp.Ordered](maximum V) Validator[V] {
	return func(value V) error {
		if !(value <= maximum) {
			return ValidationError{
				Value:  fmt.Sprint(value),
				Reason: fmt.Errorf("must be less than or equal to %v", maximum),
			}
		}

		return nil
	}
}

// Between rejects values outside the inclusive minimum and maximum bounds.
func Between[V cmp.Ordered](minimum, maximum V) Validator[V] {
	return func(value V) error {
		if !(value >= minimum && value <= maximum) {
			return ValidationError{
				Value:  fmt.Sprint(value),
				Reason: fmt.Errorf("must be between %v and %v", minimum, maximum),
			}
		}

		return nil
	}
}

// MinLen rejects strings shorter than minimum bytes.
func MinLen[S ~string](minimum int) Validator[S] {
	return func(value S) error {
		return validateMinLength(len(value), minimum)
	}
}

// MaxLen rejects strings longer than maximum bytes.
func MaxLen[S ~string](maximum int) Validator[S] {
	return func(value S) error {
		return validateMaxLength(len(value), maximum)
	}
}

// OneOf rejects values not equal to one of allowed. An empty allowed set rejects
// every value.
func OneOf[V comparable](allowed ...V) Validator[V] {
	return func(value V) error {
		for _, candidate := range allowed {
			if value == candidate {
				return nil
			}
		}

		return ValidationError{
			Value:  fmt.Sprint(value),
			Reason: fmt.Errorf("must be one of %v", allowed),
		}
	}
}

// NotOneOf rejects values equal to one of disallowed. An empty disallowed set
// accepts every value.
func NotOneOf[V comparable](disallowed ...V) Validator[V] {
	return func(value V) error {
		for _, candidate := range disallowed {
			if value == candidate {
				return ValidationError{
					Value:  fmt.Sprint(value),
					Reason: fmt.Errorf("must not be one of %v", disallowed),
				}
			}
		}

		return nil
	}
}

// SliceNotEmpty rejects nil and empty slices.
func SliceNotEmpty[S ~[]E, E any]() Validator[S] {
	return func(value S) error {
		return validateNotEmptyLength(len(value))
	}
}

// SliceMinLen rejects slices shorter than minimum.
func SliceMinLen[S ~[]E, E any](minimum int) Validator[S] {
	return func(value S) error {
		return validateMinLength(len(value), minimum)
	}
}

// SliceMaxLen rejects slices longer than maximum.
func SliceMaxLen[S ~[]E, E any](maximum int) Validator[S] {
	return func(value S) error {
		return validateMaxLength(len(value), maximum)
	}
}

// SliceEach runs every non-nil validator against every element, in ascending
// index order and validator registration order. It copies the validator list;
// nil and empty slices and zero or all-nil validators pass. Failures are joined
// under relative [index] labels, preserving each child error and its value
// details. Collection wrappers omit their own values. Traversal is read-only;
// child validators must also treat their inputs as read-only.
func SliceEach[S ~[]E, E any](validators ...Validator[E]) Validator[S] {
	validators = append([]Validator[E](nil), validators...)

	return func(value S) error {
		var failures []error

		for index, element := range value {
			for _, validator := range validators {
				if validator == nil {
					continue
				}

				if err := validator(element); err != nil {
					failures = append(failures, ValidationError{
						Field:     fmt.Sprintf("[%d]", index),
						OmitValue: true,
						Reason:    err,
					})
				}
			}
		}

		return joinCollectionFailures(failures)
	}
}

// MapNotEmpty rejects nil and empty maps.
func MapNotEmpty[M ~map[K]V, K comparable, V any]() Validator[M] {
	return func(value M) error {
		return validateNotEmptyLength(len(value))
	}
}

// MapMinLen rejects maps with fewer than minimum entries.
func MapMinLen[M ~map[K]V, K comparable, V any](minimum int) Validator[M] {
	return func(value M) error {
		return validateMinLength(len(value), minimum)
	}
}

// MapMaxLen rejects maps with more than maximum entries.
func MapMaxLen[M ~map[K]V, K comparable, V any](maximum int) Validator[M] {
	return func(value M) error {
		return validateMaxLength(len(value), maximum)
	}
}

// MapKeys runs every non-nil validator against every key, preserving validator
// registration order within each entry. Entry execution and error order are
// unspecified. It copies the validator list; nil and empty maps and zero or
// all-nil validators pass. Failures are joined under key[%#v] labels, using
// Go-syntax-style key formatting (quoted and escaped for ordinary strings) only
// for failing entries. Labels are diagnostic text, not machine-readable paths.
// Child errors and their value details are preserved; collection wrappers omit
// their own values. Traversal and child validators must be read-only.
func MapKeys[M ~map[K]V, K comparable, V any](validators ...Validator[K]) Validator[M] {
	validators = append([]Validator[K](nil), validators...)

	return func(value M) error {
		var failures []error

		for key := range value {
			var location string

			for _, validator := range validators {
				if validator == nil {
					continue
				}

				if err := validator(key); err != nil {
					if location == "" {
						location = fmt.Sprintf("key[%#v]", key)
					}

					failures = append(failures, ValidationError{
						Field:     location,
						OmitValue: true,
						Reason:    err,
					})
				}
			}
		}

		return joinCollectionFailures(failures)
	}
}

// MapValues runs every non-nil validator against every value, preserving
// validator registration order within each entry. Entry execution and error
// order are unspecified. It copies the validator list; nil and empty maps and
// zero or all-nil validators pass. Failures are joined under [%#v] key labels,
// using Go-syntax-style formatting (quoted and escaped for ordinary strings)
// only for failing entries. Labels are diagnostic text, not machine-readable
// paths. Child errors and their value details are preserved; collection wrappers
// omit their own values. Traversal and child validators must be read-only.
func MapValues[M ~map[K]V, K comparable, V any](validators ...Validator[V]) Validator[M] {
	validators = append([]Validator[V](nil), validators...)

	return func(value M) error {
		var failures []error

		for key, element := range value {
			var location string

			for _, validator := range validators {
				if validator == nil {
					continue
				}

				if err := validator(element); err != nil {
					if location == "" {
						location = fmt.Sprintf("[%#v]", key)
					}

					failures = append(failures, ValidationError{
						Field:     location,
						OmitValue: true,
						Reason:    err,
					})
				}
			}
		}

		return joinCollectionFailures(failures)
	}
}

func joinCollectionFailures(failures []error) error {
	if err := errors.Join(failures...); err != nil {
		return ValidationError{OmitValue: true, Reason: err}
	}

	return nil
}

func validateNotEmptyLength(length int) error {
	if length == 0 {
		return ValidationError{
			Reason: errors.New("must not be empty"),
		}
	}

	return nil
}

func validateMinLength(length, minimum int) error {
	if length < minimum {
		return ValidationError{
			Reason: fmt.Errorf("length must be greater than or equal to %d", minimum),
		}
	}

	return nil
}

func validateMaxLength(length, maximum int) error {
	if length > maximum {
		return ValidationError{
			Reason: fmt.Errorf("length must be less than or equal to %d", maximum),
		}
	}

	return nil
}
