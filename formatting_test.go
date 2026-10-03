package options

import (
	"errors"
	"fmt"
	"strconv"
	"testing"
)

type formattingValueSpy struct {
	value           int
	calls           *int
	collectionCalls *int
}

func (value formattingValueSpy) String() string {
	(*value.calls)++
	return "fallback"
}

type formattingSliceSpy []formattingValueSpy

func (value formattingSliceSpy) String() string {
	(*value[0].collectionCalls)++
	return "whole slice"
}

type formattingMapSpy map[string]formattingValueSpy

func (value formattingMapSpy) String() string {
	for _, element := range value {
		(*element.collectionCalls)++
		break
	}
	return "whole map"
}

func TestBuilderLazyValidationFormatting(t *testing.T) {
	cause := &validationReason{message: "invalid"}
	for _, field := range []string{"", "child"} {
		for _, value := range []string{"", "explicit"} {
			for _, omit := range []bool{false, true} {
				for _, pointer := range []bool{false, true} {
					name := fmt.Sprintf("field=%q/value=%q/omit=%t/pointer=%t", field, value, omit, pointer)
					t.Run(name, func(t *testing.T) {
						calls := 0
						original := ValidationError{Field: field, Value: value, Reason: cause, OmitValue: omit}
						owned := original
						var child error = original
						if pointer {
							child = &owned
						}
						option := New(func(_ *struct{}, _ formattingValueSpy) { t.Fatal("setter ran") }).
							Value(formattingValueSpy{calls: &calls}).Named("option").
							Validators(func(formattingValueSpy) error { return child }).Build()
						_, err := Apply(option)
						wantCalls := 0
						if !omit && (field != "" || value == "") {
							wantCalls = 1
						}
						if calls != wantCalls {
							t.Fatalf("fallback formatter calls = %d, want %d", calls, wantCalls)
						}
						want := original
						want.Field = "option"
						if field != "" {
							want = ValidationError{Field: "option", Reason: child, OmitValue: omit}
						}
						if wantCalls != 0 {
							want.Value = "fallback"
						}
						var got ValidationError
						if pointer && field == "" {
							var normalized *ValidationError
							if !errors.As(err, &normalized) || normalized == &owned {
								t.Fatal("expected a copied pointer validation error")
							}
							got = *normalized
						} else if !errors.As(err, &got) {
							t.Fatalf("expected ValidationError, got %T", err)
						}
						if !sameValidationError(got, want) || got.Reason != want.Reason || err.Error() != want.Error() {
							t.Fatalf("error = %v, want %v", err, want)
						}
						if owned != original || !errors.Is(err, cause) {
							t.Fatal("normalization mutated the child or lost its cause")
						}
					})
				}
			}
		}
	}
}

func TestBuilderOpaqueErrorFormatting(t *testing.T) {
	failure := errors.New("invalid")
	omitted := &ValidationError{OmitValue: true, Reason: failure}
	var nilPointer *ValidationError
	for _, test := range []struct {
		name string
		err  error
	}{
		{"ordinary", failure},
		{"wrapped omission", fmt.Errorf("context: %w", omitted)},
		{"joined omission", errors.Join(omitted, failure)},
		{"nil pointer", nilPointer},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			option := New(func(_ *struct{}, _ formattingValueSpy) { t.Fatal("setter ran") }).
				Value(formattingValueSpy{calls: &calls}).Named("option").
				Validators(func(formattingValueSpy) error { return test.err }).Build()
			_, err := Apply(option)
			if calls != 1 {
				t.Fatalf("fallback formatter calls = %d, want 1", calls)
			}
			var outer ValidationError
			if !errors.As(err, &outer) || outer.Field != "option" || outer.Value != "fallback" || outer.OmitValue || outer.Reason != test.err {
				t.Fatal("opaque error did not preserve legacy wrapping and fallback")
			}
		})
	}
}

func TestBuilderFallbackFormattingPerFailure(t *testing.T) {
	calls := 0
	failure := errors.New("invalid")
	option := New(func(_ *struct{}, _ formattingValueSpy) { t.Fatal("setter ran") }).
		Value(formattingValueSpy{calls: &calls}).Validators(
		func(formattingValueSpy) error { return failure },
		func(formattingValueSpy) error { return ValidationError{Value: "explicit", Reason: failure} },
		func(formattingValueSpy) error { return failure },
	).Build()
	for invocation := 1; invocation <= 2; invocation++ {
		_, err := Apply(option)
		if err == nil || calls != 2*invocation {
			t.Fatalf("invocation %d: formatter calls = %d", invocation, calls)
		}
	}
}

func TestCollectionFormattingOmission(t *testing.T) {
	failure := errors.New("reserved")
	for _, test := range []struct {
		name     string
		validate func(*testing.T, formattingValueSpy, formattingValueSpy) error
	}{
		{"slice", func(t *testing.T, invalid, valid formattingValueSpy) error {
			validator := SliceEach[formattingSliceSpy](func(value formattingValueSpy) error {
				if value.value < 0 {
					return failure
				}
				return nil
			})
			return namedCollectionError(t, formattingSliceSpy{invalid, valid}, validator)
		}},
		{"map keys", func(t *testing.T, invalid, valid formattingValueSpy) error {
			validator := MapKeys[formattingMapSpy](func(key string) error {
				if key == "invalid" {
					return failure
				}
				return nil
			})
			return namedCollectionError(t, formattingMapSpy{"invalid": invalid, "valid": valid}, validator)
		}},
		{"map values", func(t *testing.T, invalid, valid formattingValueSpy) error {
			validator := MapValues[formattingMapSpy](func(value formattingValueSpy) error {
				if value.value < 0 {
					return failure
				}
				return nil
			})
			return namedCollectionError(t, formattingMapSpy{"invalid": invalid, "valid": valid}, validator)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			collectionCalls, invalidCalls, validCalls := 0, 0, 0
			invalid := formattingValueSpy{value: -1, calls: &invalidCalls, collectionCalls: &collectionCalls}
			valid := formattingValueSpy{value: 1, calls: &validCalls, collectionCalls: &collectionCalls}
			err := test.validate(t, invalid, valid)
			if err == nil || !errors.Is(err, failure) {
				t.Fatal("expected the original failure")
			}
			_ = err.Error()
			if collectionCalls != 0 || invalidCalls != 0 || validCalls != 0 {
				t.Fatalf("formatter calls: collection=%d, invalid=%d, valid=%d", collectionCalls, invalidCalls, validCalls)
			}
		})
	}
}

type formattingKeySpy struct {
	label string
	calls *int
}

func (key formattingKeySpy) GoString() string {
	(*key.calls)++
	return strconv.Quote(key.label)
}

func TestMapLocationFormatting(t *testing.T) {
	failure := errors.New("invalid")
	for _, test := range []struct {
		name        string
		constructor func(Validator[formattingKeySpy], Validator[int]) Validator[map[formattingKeySpy]int]
		prefix      string
	}{
		{"keys", func(keys Validator[formattingKeySpy], _ Validator[int]) Validator[map[formattingKeySpy]int] {
			return MapKeys[map[formattingKeySpy]int](keys, nil, keys)
		}, "key"},
		{"values", func(_ Validator[formattingKeySpy], values Validator[int]) Validator[map[formattingKeySpy]int] {
			return MapValues[map[formattingKeySpy]int](values, nil, values)
		}, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			validCalls, invalidCalls := 0, 0
			valid := formattingKeySpy{label: "valid", calls: &validCalls}
			invalid := formattingKeySpy{label: "a\"b\n", calls: &invalidCalls}
			validator := test.constructor(
				func(key formattingKeySpy) error {
					if key.label != "valid" {
						return failure
					}
					return nil
				},
				func(value int) error {
					if value < 0 {
						return failure
					}
					return nil
				},
			)
			if err := validator(map[formattingKeySpy]int{valid: 1}); err != nil || validCalls != 0 {
				t.Fatal("successful key was formatted or failed validation")
			}
			for invocation := 1; invocation <= 2; invocation++ {
				err := validator(map[formattingKeySpy]int{valid: 1, invalid: -1})
				failures := locatedCollectionFailures(t, err)
				_ = err.Error()
				if validCalls != 0 || invalidCalls != invocation || len(failures) != 2 {
					t.Fatalf("formatter calls: valid=%d, invalid=%d; failures=%d", validCalls, invalidCalls, len(failures))
				}
				for _, located := range failures {
					if located.Field != test.prefix+`["a\"b\n"]` || located.Reason != failure {
						t.Fatalf("location = %q", located.Field)
					}
				}
			}
		})
	}
}
