package options

import (
	"cmp"
	"errors"
	"fmt"
	"reflect"
	"strconv"
)

// Validator validates a value. It returns nil when the value is valid and an
// error describing why validation failed otherwise. Validators should treat the
// value as read-only. Use errors.Join to return multiple failures.
type Validator[V any] func(V) error

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

// Min rejects values smaller than minimum.
func Min[V cmp.Ordered](minimum V) Validator[V] {
	return func(value V) error {
		if value < minimum {
			return ValidationError{
				Value:  fmt.Sprint(value),
				Reason: fmt.Errorf("must be greater than or equal to %v", minimum),
			}
		}

		return nil
	}
}

// Max rejects values larger than maximum.
func Max[V cmp.Ordered](maximum V) Validator[V] {
	return func(value V) error {
		if value > maximum {
			return ValidationError{
				Value:  fmt.Sprint(value),
				Reason: fmt.Errorf("must be less than or equal to %v", maximum),
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
