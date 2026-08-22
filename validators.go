package options

import (
	"cmp"
	"fmt"
	"reflect"
	"strconv"
)

// Validator validates a value and reports each validation failure it finds.
// Validators should treat the value as read-only.
type Validator[V any] func(V, Report)

// Check adapts check into a Validator. A check may report multiple failures.
func Check[V any](check func(V, Report)) Validator[V] {
	return check
}

// NotNil rejects nil values, including typed nil values stored in interfaces.
// Values whose type cannot be nil always pass validation.
func NotNil[V any]() Validator[V] {
	return func(value V, report Report) {
		reflected := reflect.ValueOf(value)
		if !reflected.IsValid() {
			report(ValidationError{
				Value:  "<nil>",
				Reason: "must not be nil",
			})
			return
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
				report(ValidationError{
					Value:  "<nil>",
					Reason: "must not be nil",
				})
			}
		}
	}
}

// NotNilPtr rejects nil pointers without using reflection. Use NotNil when the
// value may be another nil-capable type.
func NotNilPtr[V any]() Validator[*V] {
	return func(value *V, report Report) {
		if value == nil {
			report(ValidationError{Reason: "cannot be nil"})
		}
	}
}

// NotZero rejects the zero value of V.
func NotZero[V comparable]() Validator[V] {
	return func(value V, report Report) {
		var zero V
		if value == zero {
			report(ValidationError{
				Value:  fmt.Sprint(value),
				Reason: "must not be zero",
			})
		}
	}
}

// NotEmpty rejects empty string values.
func NotEmpty[S ~string]() Validator[S] {
	return func(value S, report Report) {
		if value == "" {
			report(ValidationError{
				Value:  strconv.Quote(string(value)),
				Reason: "must not be empty",
			})
		}
	}
}

// Min rejects values smaller than minimum.
func Min[V cmp.Ordered](minimum V) Validator[V] {
	return func(value V, report Report) {
		if value < minimum {
			report(ValidationError{
				Value:  fmt.Sprint(value),
				Reason: fmt.Sprintf("must be greater than or equal to %v", minimum),
			})
		}
	}
}

// Max rejects values larger than maximum.
func Max[V cmp.Ordered](maximum V) Validator[V] {
	return func(value V, report Report) {
		if value > maximum {
			report(ValidationError{
				Value:  fmt.Sprint(value),
				Reason: fmt.Sprintf("must be less than or equal to %v", maximum),
			})
		}
	}
}

// MinLen rejects strings shorter than minimum bytes.
func MinLen[S ~string](minimum int) Validator[S] {
	return func(value S, report Report) {
		reportMinLength(len(value), minimum, report)
	}
}

// MaxLen rejects strings longer than maximum bytes.
func MaxLen[S ~string](maximum int) Validator[S] {
	return func(value S, report Report) {
		reportMaxLength(len(value), maximum, report)
	}
}

// OneOf rejects values not equal to one of allowed. An empty allowed set rejects
// every value.
func OneOf[V comparable](allowed ...V) Validator[V] {
	return func(value V, report Report) {
		for _, candidate := range allowed {
			if value == candidate {
				return
			}
		}

		report(ValidationError{
			Value:  fmt.Sprint(value),
			Reason: fmt.Sprintf("must be one of %v", allowed),
		})
	}
}

// SliceNotEmpty rejects nil and empty slices.
func SliceNotEmpty[S ~[]E, E any]() Validator[S] {
	return func(value S, report Report) {
		reportNotEmptyLength(len(value), report)
	}
}

// SliceMinLen rejects slices shorter than minimum.
func SliceMinLen[S ~[]E, E any](minimum int) Validator[S] {
	return func(value S, report Report) {
		reportMinLength(len(value), minimum, report)
	}
}

// SliceMaxLen rejects slices longer than maximum.
func SliceMaxLen[S ~[]E, E any](maximum int) Validator[S] {
	return func(value S, report Report) {
		reportMaxLength(len(value), maximum, report)
	}
}

// MapNotEmpty rejects nil and empty maps.
func MapNotEmpty[M ~map[K]V, K comparable, V any]() Validator[M] {
	return func(value M, report Report) {
		reportNotEmptyLength(len(value), report)
	}
}

// MapMinLen rejects maps with fewer than minimum entries.
func MapMinLen[M ~map[K]V, K comparable, V any](minimum int) Validator[M] {
	return func(value M, report Report) {
		reportMinLength(len(value), minimum, report)
	}
}

// MapMaxLen rejects maps with more than maximum entries.
func MapMaxLen[M ~map[K]V, K comparable, V any](maximum int) Validator[M] {
	return func(value M, report Report) {
		reportMaxLength(len(value), maximum, report)
	}
}

func reportNotEmptyLength(length int, report Report) {
	if length == 0 {
		report(ValidationError{
			Reason: "must not be empty",
		})
	}
}

func reportMinLength(length, minimum int, report Report) {
	if length < minimum {
		report(ValidationError{
			Reason: fmt.Sprintf("length must be greater than or equal to %d", minimum),
		})
	}
}

func reportMaxLength(length, maximum int, report Report) {
	if length > maximum {
		report(ValidationError{
			Reason: fmt.Sprintf("length must be less than or equal to %d", maximum),
		})
	}
}
