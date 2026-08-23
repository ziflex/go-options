package options

import (
	"bytes"
	"errors"
	"io"
	"math"
	"testing"
	"unsafe"
)

func validationFailures[V any](value V, validator Validator[V]) []ValidationError {
	err := validator(value)
	if err == nil {
		return nil
	}

	var failure ValidationError
	if !errors.As(err, &failure) {
		return nil
	}

	return []ValidationError{failure}
}

func TestCheck(t *testing.T) {
	first := errors.New("first")
	second := errors.New("second")
	joined := errors.Join(first, second)
	validator := Check(func(value int) error {
		if value < 0 {
			return joined
		}

		return nil
	})

	if err := validator(0); err != nil {
		t.Fatalf("validator(0) error = %v, want nil", err)
	}
	if err := validator(-1); err != joined {
		t.Fatalf("validator(-1) error = %v, want original joined error", err)
	}

	var nilCheck func(int) error
	if validator := Check(nilCheck); validator != nil {
		t.Fatal("Check(nil) returned a non-nil validator")
	}
}

func TestNotNil(t *testing.T) {
	wantFailure := ValidationError{Value: "<nil>", Reason: errors.New("must not be nil")}
	assertValid := func(t *testing.T, failures []ValidationError) {
		t.Helper()
		if len(failures) != 0 {
			t.Fatalf("failures = %+v, want none", failures)
		}
	}
	assertNil := func(t *testing.T, failures []ValidationError) {
		t.Helper()
		if len(failures) != 1 {
			t.Fatalf("failure count = %d, want 1: %+v", len(failures), failures)
		}
		if !sameValidationError(failures[0], wantFailure) {
			t.Fatalf("failure = %+v, want %+v", failures[0], wantFailure)
		}
	}

	t.Run("pointer", func(t *testing.T) {
		var nilPointer *int
		assertNil(t, validationFailures(nilPointer, NotNil[*int]()))

		value := 1
		assertValid(t, validationFailures(&value, NotNil[*int]()))
	})

	t.Run("interface", func(t *testing.T) {
		var nilWriter io.Writer
		assertNil(t, validationFailures(nilWriter, NotNil[io.Writer]()))

		var writer io.Writer = &bytes.Buffer{}
		assertValid(t, validationFailures(writer, NotNil[io.Writer]()))
	})

	t.Run("typed nil in interface", func(t *testing.T) {
		var writer io.Writer = (*bytes.Buffer)(nil)
		assertNil(t, validationFailures(writer, NotNil[io.Writer]()))
	})

	t.Run("slice", func(t *testing.T) {
		var nilSlice []int
		assertNil(t, validationFailures(nilSlice, NotNil[[]int]()))
		assertValid(t, validationFailures([]int{}, NotNil[[]int]()))
	})

	t.Run("map", func(t *testing.T) {
		var nilMap map[string]int
		assertNil(t, validationFailures(nilMap, NotNil[map[string]int]()))
		assertValid(t, validationFailures(map[string]int{}, NotNil[map[string]int]()))
	})

	t.Run("function", func(t *testing.T) {
		var nilFunction func()
		assertNil(t, validationFailures(nilFunction, NotNil[func()]()))
		assertValid(t, validationFailures(func() {}, NotNil[func()]()))
	})

	t.Run("channel", func(t *testing.T) {
		var nilChannel chan int
		assertNil(t, validationFailures(nilChannel, NotNil[chan int]()))

		channel := make(chan int)
		defer close(channel)
		assertValid(t, validationFailures(channel, NotNil[chan int]()))
	})

	t.Run("unsafe pointer", func(t *testing.T) {
		var nilPointer unsafe.Pointer
		assertNil(t, validationFailures(nilPointer, NotNil[unsafe.Pointer]()))

		value := 1
		assertValid(t, validationFailures(unsafe.Pointer(&value), NotNil[unsafe.Pointer]()))
	})

	t.Run("non-nilable values", func(t *testing.T) {
		assertValid(t, validationFailures(0, NotNil[int]()))
		assertValid(t, validationFailures(false, NotNil[bool]()))
		assertValid(t, validationFailures(struct{}{}, NotNil[struct{}]()))
	})
}

func TestNotNilPtr(t *testing.T) {
	var validator Validator[*int] = NotNilPtr[int]()

	t.Run("nil pointer", func(t *testing.T) {
		failures := validationFailures((*int)(nil), validator)
		if len(failures) != 1 {
			t.Fatalf("failure count = %d, want 1: %+v", len(failures), failures)
		}
		want := ValidationError{Reason: errors.New("cannot be nil")}
		if !sameValidationError(failures[0], want) {
			t.Fatalf("failure = %+v, want %+v", failures[0], want)
		}
	})

	t.Run("non-nil pointer", func(t *testing.T) {
		value := 1
		if failures := validationFailures(&value, validator); len(failures) != 0 {
			t.Fatalf("failures = %+v, want none", failures)
		}
	})
}

func TestNotZero(t *testing.T) {
	type count int

	if failures := validationFailures(count(1), NotZero[count]()); len(failures) != 0 {
		t.Fatalf("non-zero failures = %+v", failures)
	}
	failures := validationFailures(count(0), NotZero[count]())
	if len(failures) != 1 || errorMessage(failures[0].Reason) != "must not be zero" {
		t.Fatalf("zero failures = %+v", failures)
	}

	value := 1
	if failures := validationFailures(&value, NotZero[*int]()); len(failures) != 0 {
		t.Fatalf("non-nil pointer failures = %+v", failures)
	}
	if failures := validationFailures((*int)(nil), NotZero[*int]()); len(failures) != 1 {
		t.Fatalf("nil pointer failure count = %d, want 1", len(failures))
	}
}

func TestNumericSignValidators(t *testing.T) {
	type count int

	tests := []struct {
		name       string
		value      count
		validator  Validator[count]
		wantValue  string
		wantReason string
	}{
		{name: "positive negative", value: -1, validator: Positive[count](), wantValue: "-1", wantReason: "must be positive"},
		{name: "positive zero", value: 0, validator: Positive[count](), wantValue: "0", wantReason: "must be positive"},
		{name: "positive positive", value: 1, validator: Positive[count]()},
		{name: "non-negative negative", value: -1, validator: NonNegative[count](), wantValue: "-1", wantReason: "must be non-negative"},
		{name: "non-negative zero", value: 0, validator: NonNegative[count]()},
		{name: "non-negative positive", value: 1, validator: NonNegative[count]()},
		{name: "negative negative", value: -1, validator: Negative[count]()},
		{name: "negative zero", value: 0, validator: Negative[count](), wantValue: "0", wantReason: "must be negative"},
		{name: "negative positive", value: 1, validator: Negative[count](), wantValue: "1", wantReason: "must be negative"},
		{name: "non-positive negative", value: -1, validator: NonPositive[count]()},
		{name: "non-positive zero", value: 0, validator: NonPositive[count]()},
		{name: "non-positive positive", value: 1, validator: NonPositive[count](), wantValue: "1", wantReason: "must be non-positive"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			failures := validationFailures(test.value, test.validator)
			if test.wantReason == "" {
				if len(failures) != 0 {
					t.Fatalf("failures = %+v, want none", failures)
				}

				return
			}

			if len(failures) != 1 {
				t.Fatalf("failure count = %d, want 1: %+v", len(failures), failures)
			}
			want := ValidationError{Value: test.wantValue, Reason: errors.New(test.wantReason)}
			if !sameValidationError(failures[0], want) {
				t.Fatalf("failure = %+v, want %+v", failures[0], want)
			}
		})
	}
}

func TestNumericSignValidatorsUnsigned(t *testing.T) {
	tests := []struct {
		name      string
		value     uint
		validator Validator[uint]
		wantFail  bool
	}{
		{name: "positive zero", value: 0, validator: Positive[uint](), wantFail: true},
		{name: "positive nonzero", value: 1, validator: Positive[uint]()},
		{name: "non-negative zero", value: 0, validator: NonNegative[uint]()},
		{name: "non-negative nonzero", value: 1, validator: NonNegative[uint]()},
		{name: "negative zero", value: 0, validator: Negative[uint](), wantFail: true},
		{name: "negative nonzero", value: 1, validator: Negative[uint](), wantFail: true},
		{name: "non-positive zero", value: 0, validator: NonPositive[uint]()},
		{name: "non-positive nonzero", value: 1, validator: NonPositive[uint](), wantFail: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			failures := validationFailures(test.value, test.validator)
			if got := len(failures) == 1; got != test.wantFail {
				t.Fatalf("failures = %+v, wantFail = %v", failures, test.wantFail)
			}
		})
	}
}

func TestNumericSignValidatorsFloat(t *testing.T) {
	negativeZero := math.Copysign(0, -1)
	tests := []struct {
		name      string
		value     float64
		validator Validator[float64]
		wantFail  bool
	}{
		{name: "positive negative infinity", value: math.Inf(-1), validator: Positive[float64](), wantFail: true},
		{name: "positive negative finite", value: -1.5, validator: Positive[float64](), wantFail: true},
		{name: "positive negative zero", value: negativeZero, validator: Positive[float64](), wantFail: true},
		{name: "positive positive finite", value: 1.5, validator: Positive[float64]()},
		{name: "positive positive infinity", value: math.Inf(1), validator: Positive[float64]()},
		{name: "non-negative negative infinity", value: math.Inf(-1), validator: NonNegative[float64](), wantFail: true},
		{name: "non-negative negative finite", value: -1.5, validator: NonNegative[float64](), wantFail: true},
		{name: "non-negative negative zero", value: negativeZero, validator: NonNegative[float64]()},
		{name: "non-negative positive finite", value: 1.5, validator: NonNegative[float64]()},
		{name: "non-negative positive infinity", value: math.Inf(1), validator: NonNegative[float64]()},
		{name: "negative negative infinity", value: math.Inf(-1), validator: Negative[float64]()},
		{name: "negative negative finite", value: -1.5, validator: Negative[float64]()},
		{name: "negative negative zero", value: negativeZero, validator: Negative[float64](), wantFail: true},
		{name: "negative positive finite", value: 1.5, validator: Negative[float64](), wantFail: true},
		{name: "negative positive infinity", value: math.Inf(1), validator: Negative[float64](), wantFail: true},
		{name: "non-positive negative infinity", value: math.Inf(-1), validator: NonPositive[float64]()},
		{name: "non-positive negative finite", value: -1.5, validator: NonPositive[float64]()},
		{name: "non-positive negative zero", value: negativeZero, validator: NonPositive[float64]()},
		{name: "non-positive positive finite", value: 1.5, validator: NonPositive[float64](), wantFail: true},
		{name: "non-positive positive infinity", value: math.Inf(1), validator: NonPositive[float64](), wantFail: true},
		{name: "positive NaN", value: math.NaN(), validator: Positive[float64](), wantFail: true},
		{name: "non-negative NaN", value: math.NaN(), validator: NonNegative[float64](), wantFail: true},
		{name: "negative NaN", value: math.NaN(), validator: Negative[float64](), wantFail: true},
		{name: "non-positive NaN", value: math.NaN(), validator: NonPositive[float64](), wantFail: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			failures := validationFailures(test.value, test.validator)
			if got := len(failures) == 1; got != test.wantFail {
				t.Fatalf("failures = %+v, wantFail = %v", failures, test.wantFail)
			}
		})
	}
}

func TestNotEmpty(t *testing.T) {
	type name string

	if failures := validationFailures(name("service"), NotEmpty[name]()); len(failures) != 0 {
		t.Fatalf("non-empty failures = %+v", failures)
	}
	failures := validationFailures(name(""), NotEmpty[name]())
	if len(failures) != 1 {
		t.Fatalf("empty failure count = %d, want 1", len(failures))
	}
	if failures[0].Value != `""` || errorMessage(failures[0].Reason) != "must not be empty" {
		t.Fatalf("failure = %+v", failures[0])
	}
}

func TestNotBlank(t *testing.T) {
	type name string

	tests := []struct {
		name     string
		value    name
		wantFail bool
	}{
		{name: "empty", value: "", wantFail: true},
		{name: "space", value: " ", wantFail: true},
		{name: "ASCII whitespace", value: "\t\n\r", wantFail: true},
		{name: "Unicode whitespace", value: "\u2003", wantFail: true},
		{name: "text", value: "service"},
		{name: "surrounding whitespace", value: "\t service \u2003"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			failures := validationFailures(test.value, NotBlank[name]())
			if got := len(failures) == 1; got != test.wantFail {
				t.Fatalf("failures = %+v, wantFail = %v", failures, test.wantFail)
			}
		})
	}

	failures := validationFailures(name(" \t"), NotBlank[name]())
	want := ValidationError{Value: `" \t"`, Reason: errors.New("must not be blank")}
	if len(failures) != 1 || !sameValidationError(failures[0], want) {
		t.Fatalf("failures = %+v, want [%+v]", failures, want)
	}
}

func TestMinAndMax(t *testing.T) {
	type count int

	tests := []struct {
		name      string
		value     count
		validator Validator[count]
		wantFail  bool
	}{
		{name: "min below", value: 1, validator: Min(count(2)), wantFail: true},
		{name: "min equal", value: 2, validator: Min(count(2))},
		{name: "min above", value: 3, validator: Min(count(2))},
		{name: "max below", value: 1, validator: Max(count(2))},
		{name: "max equal", value: 2, validator: Max(count(2))},
		{name: "max above", value: 3, validator: Max(count(2)), wantFail: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			failures := validationFailures(test.value, test.validator)
			if got := len(failures) == 1; got != test.wantFail {
				t.Fatalf("failures = %+v, wantFail = %v", failures, test.wantFail)
			}
		})
	}

	if failures := validationFailures("a", Min("b")); len(failures) != 1 {
		t.Fatalf("ordered string failure count = %d, want 1", len(failures))
	}

	minFailure := validationFailures(count(1), Min(count(2)))
	wantMinFailure := ValidationError{Value: "1", Reason: errors.New("must be greater than or equal to 2")}
	if len(minFailure) != 1 || !sameValidationError(minFailure[0], wantMinFailure) {
		t.Fatalf("Min failure = %+v, want [%+v]", minFailure, wantMinFailure)
	}
	maxFailure := validationFailures(count(3), Max(count(2)))
	wantMaxFailure := ValidationError{Value: "3", Reason: errors.New("must be less than or equal to 2")}
	if len(maxFailure) != 1 || !sameValidationError(maxFailure[0], wantMaxFailure) {
		t.Fatalf("Max failure = %+v, want [%+v]", maxFailure, wantMaxFailure)
	}

	floatTests := []struct {
		name       string
		value      float64
		validator  Validator[float64]
		wantValue  string
		wantReason string
	}{
		{name: "Min NaN value", value: math.NaN(), validator: Min(0.0), wantValue: "NaN", wantReason: "must be greater than or equal to 0"},
		{name: "Min NaN bound", value: 0, validator: Min(math.NaN()), wantValue: "0", wantReason: "must be greater than or equal to NaN"},
		{name: "Max NaN value", value: math.NaN(), validator: Max(0.0), wantValue: "NaN", wantReason: "must be less than or equal to 0"},
		{name: "Max NaN bound", value: 0, validator: Max(math.NaN()), wantValue: "0", wantReason: "must be less than or equal to NaN"},
	}
	for _, test := range floatTests {
		t.Run(test.name, func(t *testing.T) {
			failures := validationFailures(test.value, test.validator)
			want := ValidationError{Value: test.wantValue, Reason: errors.New(test.wantReason)}
			if len(failures) != 1 || !sameValidationError(failures[0], want) {
				t.Fatalf("failures = %+v, want [%+v]", failures, want)
			}
		})
	}
}

func TestBetween(t *testing.T) {
	type count int

	tests := []struct {
		name     string
		value    count
		minimum  count
		maximum  count
		wantFail bool
	}{
		{name: "below", value: 0, minimum: 1, maximum: 3, wantFail: true},
		{name: "minimum", value: 1, minimum: 1, maximum: 3},
		{name: "within", value: 2, minimum: 1, maximum: 3},
		{name: "maximum", value: 3, minimum: 1, maximum: 3},
		{name: "above", value: 4, minimum: 1, maximum: 3, wantFail: true},
		{name: "reversed below", value: 0, minimum: 3, maximum: 1, wantFail: true},
		{name: "reversed within", value: 2, minimum: 3, maximum: 1, wantFail: true},
		{name: "reversed above", value: 4, minimum: 3, maximum: 1, wantFail: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			failures := validationFailures(test.value, Between(test.minimum, test.maximum))
			if got := len(failures) == 1; got != test.wantFail {
				t.Fatalf("failures = %+v, wantFail = %v", failures, test.wantFail)
			}
		})
	}

	if failures := validationFailures("c", Between("b", "d")); len(failures) != 0 {
		t.Fatalf("ordered string failures = %+v, want none", failures)
	}
	if failures := validationFailures(math.Inf(-1), Between(math.Inf(-1), math.Inf(1))); len(failures) != 0 {
		t.Fatalf("negative infinity failures = %+v, want none", failures)
	}
	if failures := validationFailures(math.Inf(1), Between(math.Inf(-1), math.Inf(1))); len(failures) != 0 {
		t.Fatalf("positive infinity failures = %+v, want none", failures)
	}

	floatTests := []struct {
		name    string
		value   float64
		minimum float64
		maximum float64
	}{
		{name: "NaN value", value: math.NaN(), minimum: 0, maximum: 1},
		{name: "NaN minimum", value: 0.5, minimum: math.NaN(), maximum: 1},
		{name: "NaN maximum", value: 0.5, minimum: 0, maximum: math.NaN()},
	}
	for _, test := range floatTests {
		t.Run(test.name, func(t *testing.T) {
			if failures := validationFailures(test.value, Between(test.minimum, test.maximum)); len(failures) != 1 {
				t.Fatalf("failure count = %d, want 1: %+v", len(failures), failures)
			}
		})
	}

	failures := validationFailures(count(4), Between(count(1), count(3)))
	want := ValidationError{Value: "4", Reason: errors.New("must be between 1 and 3")}
	if len(failures) != 1 || !sameValidationError(failures[0], want) {
		t.Fatalf("failures = %+v, want [%+v]", failures, want)
	}
}

func TestStringLengthValidators(t *testing.T) {
	type name string

	tests := []struct {
		name      string
		value     name
		validator Validator[name]
		wantFail  bool
	}{
		{name: "min below", value: "a", validator: MinLen[name](2), wantFail: true},
		{name: "min equal", value: "ab", validator: MinLen[name](2)},
		{name: "min above", value: "abc", validator: MinLen[name](2)},
		{name: "max below", value: "a", validator: MaxLen[name](2)},
		{name: "max equal", value: "ab", validator: MaxLen[name](2)},
		{name: "max above", value: "abc", validator: MaxLen[name](2), wantFail: true},
		{name: "negative min", value: "", validator: MinLen[name](-1)},
		{name: "negative max", value: "", validator: MaxLen[name](-1), wantFail: true},
		{name: "byte length", value: "é", validator: MaxLen[name](1), wantFail: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			failures := validationFailures(test.value, test.validator)
			if got := len(failures) == 1; got != test.wantFail {
				t.Fatalf("failures = %+v, wantFail = %v", failures, test.wantFail)
			}
		})
	}

	minFailure := validationFailures(name("a"), MinLen[name](2))[0]
	if minFailure.Value != "" || errorMessage(minFailure.Reason) != "length must be greater than or equal to 2" {
		t.Fatalf("minimum-length failure = %+v", minFailure)
	}
	maxFailure := validationFailures(name("ab"), MaxLen[name](1))[0]
	if maxFailure.Value != "" || errorMessage(maxFailure.Reason) != "length must be less than or equal to 1" {
		t.Fatalf("maximum-length failure = %+v", maxFailure)
	}
}

func TestOneOf(t *testing.T) {
	type mode string

	tests := []struct {
		name     string
		value    mode
		allowed  []mode
		wantFail bool
	}{
		{name: "allowed", value: "fast", allowed: []mode{"fast", "safe"}},
		{name: "rejected", value: "other", allowed: []mode{"fast", "safe"}, wantFail: true},
		{name: "duplicate allowed", value: "fast", allowed: []mode{"fast", "fast"}},
		{name: "empty allowed set", value: "fast", wantFail: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			failures := validationFailures(test.value, OneOf(test.allowed...))
			if got := len(failures) == 1; got != test.wantFail {
				t.Fatalf("failures = %+v, wantFail = %v", failures, test.wantFail)
			}
		})
	}

	failures := validationFailures(mode("other"), OneOf(mode("fast"), mode("safe")))
	if failures[0].Value != "other" || errorMessage(failures[0].Reason) != "must be one of [fast safe]" {
		t.Fatalf("failure = %+v", failures[0])
	}
}

func TestNotOneOf(t *testing.T) {
	type mode string

	tests := []struct {
		name       string
		value      mode
		disallowed []mode
		wantFail   bool
	}{
		{name: "allowed", value: "other", disallowed: []mode{"fast", "safe"}},
		{name: "rejected", value: "fast", disallowed: []mode{"fast", "safe"}, wantFail: true},
		{name: "duplicate disallowed", value: "fast", disallowed: []mode{"fast", "fast"}, wantFail: true},
		{name: "empty disallowed set", value: "fast"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			failures := validationFailures(test.value, NotOneOf(test.disallowed...))
			if got := len(failures) == 1; got != test.wantFail {
				t.Fatalf("failures = %+v, wantFail = %v", failures, test.wantFail)
			}
		})
	}

	failures := validationFailures(mode("fast"), NotOneOf(mode("fast"), mode("safe")))
	want := ValidationError{Value: "fast", Reason: errors.New("must not be one of [fast safe]")}
	if len(failures) != 1 || !sameValidationError(failures[0], want) {
		t.Fatalf("failures = %+v, want [%+v]", failures, want)
	}
}

func TestSliceValidators(t *testing.T) {
	type identifiers []int

	tests := []struct {
		name      string
		value     identifiers
		validator Validator[identifiers]
		wantFail  bool
	}{
		{name: "not empty nil", value: nil, validator: SliceNotEmpty[identifiers](), wantFail: true},
		{name: "not empty empty", value: identifiers{}, validator: SliceNotEmpty[identifiers](), wantFail: true},
		{name: "not empty populated", value: identifiers{1}, validator: SliceNotEmpty[identifiers]()},
		{name: "min below", value: identifiers{1}, validator: SliceMinLen[identifiers](2), wantFail: true},
		{name: "min equal", value: identifiers{1, 2}, validator: SliceMinLen[identifiers](2)},
		{name: "min above", value: identifiers{1, 2, 3}, validator: SliceMinLen[identifiers](2)},
		{name: "max below", value: identifiers{1}, validator: SliceMaxLen[identifiers](2)},
		{name: "max equal", value: identifiers{1, 2}, validator: SliceMaxLen[identifiers](2)},
		{name: "max above", value: identifiers{1, 2, 3}, validator: SliceMaxLen[identifiers](2), wantFail: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			failures := validationFailures(test.value, test.validator)
			if got := len(failures) == 1; got != test.wantFail {
				t.Fatalf("failures = %+v, wantFail = %v", failures, test.wantFail)
			}
		})
	}

	diagnostics := []ValidationError{
		validationFailures(identifiers(nil), SliceNotEmpty[identifiers]())[0],
		validationFailures(identifiers{1}, SliceMinLen[identifiers](2))[0],
		validationFailures(identifiers{1, 2}, SliceMaxLen[identifiers](1))[0],
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Value != "" {
			t.Errorf("failure Value = %q, want empty: %+v", diagnostic.Value, diagnostic)
		}
	}
}

func TestMapValidators(t *testing.T) {
	type labels map[string]int

	tests := []struct {
		name      string
		value     labels
		validator Validator[labels]
		wantFail  bool
	}{
		{name: "not empty nil", value: nil, validator: MapNotEmpty[labels](), wantFail: true},
		{name: "not empty empty", value: labels{}, validator: MapNotEmpty[labels](), wantFail: true},
		{name: "not empty populated", value: labels{"a": 1}, validator: MapNotEmpty[labels]()},
		{name: "min below", value: labels{"a": 1}, validator: MapMinLen[labels](2), wantFail: true},
		{name: "min equal", value: labels{"a": 1, "b": 2}, validator: MapMinLen[labels](2)},
		{name: "min above", value: labels{"a": 1, "b": 2, "c": 3}, validator: MapMinLen[labels](2)},
		{name: "max below", value: labels{"a": 1}, validator: MapMaxLen[labels](2)},
		{name: "max equal", value: labels{"a": 1, "b": 2}, validator: MapMaxLen[labels](2)},
		{name: "max above", value: labels{"a": 1, "b": 2, "c": 3}, validator: MapMaxLen[labels](2), wantFail: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			failures := validationFailures(test.value, test.validator)
			if got := len(failures) == 1; got != test.wantFail {
				t.Fatalf("failures = %+v, wantFail = %v", failures, test.wantFail)
			}
		})
	}

	diagnostics := []ValidationError{
		validationFailures(labels(nil), MapNotEmpty[labels]())[0],
		validationFailures(labels{"a": 1}, MapMinLen[labels](2))[0],
		validationFailures(labels{"a": 1, "b": 2}, MapMaxLen[labels](1))[0],
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Value != "" {
			t.Errorf("failure Value = %q, want empty: %+v", diagnostic.Value, diagnostic)
		}
	}
}
