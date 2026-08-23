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
	if failures := validationFailures(math.NaN(), Min(0.0)); len(failures) != 0 {
		t.Fatalf("NaN Min failures = %+v", failures)
	}
	if failures := validationFailures(math.NaN(), Max(0.0)); len(failures) != 0 {
		t.Fatalf("NaN Max failures = %+v", failures)
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
