package options

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

type constructorConfig struct {
	value int
	ptr   *int
}

func setConstructorValue(config *constructorConfig, value int) {
	config.value = value
}

func TestBuilder(t *testing.T) {
	t.Run("infers types and builds without validators", func(t *testing.T) {
		calls := 0
		var option Option[constructorConfig] = New(
			func(config *constructorConfig, value int) {
				calls++
				config.value = value
			},
		).Value(8080).Build()

		var config constructorConfig
		if err := option(&config); err != nil {
			t.Fatalf("option error = %v", err)
		}
		if calls != 1 || config.value != 8080 {
			t.Fatalf("setter calls = %d, config.value = %d", calls, config.value)
		}
	})

	t.Run("explicit zero value", func(t *testing.T) {
		initial := constructorConfig{value: 9}
		calls := 0
		option := New(func(config *constructorConfig, value int) {
			calls++
			config.value = value
		}).Value(0).Build()

		config, err := ApplyTo(initial, option)
		if err != nil {
			t.Fatalf("ApplyTo() error = %v", err)
		}
		if calls != 1 || config.value != 0 {
			t.Fatalf("setter calls = %d, config.value = %d", calls, config.value)
		}
	})

	t.Run("missing value", func(t *testing.T) {
		initial := constructorConfig{value: 9}
		setterCalls := 0
		validatorCalls := 0
		option := New(func(config *constructorConfig, value int) {
			setterCalls++
			config.value = value
		}).Named("value").Validators(
			func(_ int) error {
				validatorCalls++
				return nil
			},
		).Build()

		config, err := ApplyTo(initial, option)
		if err == nil {
			t.Fatal("ApplyTo() error = nil, want missing-value error")
		}
		if got, want := err.Error(), "option value was not provided"; got != want {
			t.Fatalf("ApplyTo() error = %q, want %q", got, want)
		}
		var validationError ValidationError
		wantValidationError := ValidationError{Reason: errors.New("option value was not provided")}
		if !errors.As(err, &validationError) || !sameValidationError(validationError, wantValidationError) {
			t.Fatalf("ApplyTo() error = %v, want %+v", err, wantValidationError)
		}
		if setterCalls != 0 || validatorCalls != 0 || config != initial {
			t.Fatalf(
				"setter calls = %d, validator calls = %d, config = %+v, want untouched %+v",
				setterCalls,
				validatorCalls,
				config,
				initial,
			)
		}
	})

	t.Run("repeated Value uses latest value", func(t *testing.T) {
		option := New(setConstructorValue).Value(8080).Value(9090).Build()

		config, err := Apply(option)
		if err != nil {
			t.Fatalf("Apply() error = %v", err)
		}
		if config.value != 9090 {
			t.Fatalf("config.value = %d, want 9090", config.value)
		}
	})

	t.Run("passing validator", func(t *testing.T) {
		config, err := Apply(
			New(setConstructorValue).
				Value(7).
				Validators(Min(1)).
				Build(),
		)
		if err != nil {
			t.Fatalf("Apply() error = %v", err)
		}
		if config.value != 7 {
			t.Fatalf("config.value = %d, want 7", config.value)
		}
	})

	t.Run("failed validation prevents setter", func(t *testing.T) {
		initial := constructorConfig{value: 9}
		calls := 0
		option := New(func(config *constructorConfig, value int) {
			calls++
			config.value = value
		}).Value(0).Validators(Min(1)).Build()

		config, err := ApplyTo(initial, option)
		if err == nil {
			t.Fatal("ApplyTo() error = nil, want validation error")
		}
		if calls != 0 || config != initial {
			t.Fatalf("setter calls = %d, config = %+v, want %+v", calls, config, initial)
		}
	})

	t.Run("Validators appends and preserves order", func(t *testing.T) {
		var order []int
		option := New(setConstructorValue).
			Value(5).
			Validators(func(_ int) error {
				order = append(order, 1)
				return ValidationError{Reason: errors.New("first")}
			}).
			Validators(
				nil,
				func(_ int) error {
					order = append(order, 2)
					return nil
				},
				func(_ int) error {
					order = append(order, 3)
					return ValidationError{Reason: errors.New("third")}
				},
			).
			Build()

		config, err := Apply(option)
		if err == nil {
			t.Fatal("Apply() error = nil, want joined validation error")
		}
		if !reflect.DeepEqual(order, []int{1, 2, 3}) {
			t.Fatalf("validator order = %v, want [1 2 3]", order)
		}
		if config.value != 0 {
			t.Fatalf("config.value = %d, want zero", config.value)
		}

		applyErrors := err.(interface{ Unwrap() []error }).Unwrap()
		if len(applyErrors) != 1 {
			t.Fatalf("Apply() error count = %d, want one option error", len(applyErrors))
		}
		validatorErrors := applyErrors[0].(interface{ Unwrap() []error }).Unwrap()
		if len(validatorErrors) != 2 {
			t.Fatalf("validator error count = %d, want 2", len(validatorErrors))
		}
	})

	t.Run("Named enriches unnamed failure", func(t *testing.T) {
		_, err := Apply(
			New(setConstructorValue).
				Value(0).
				Named("old").
				Named("value").
				Validators(Min(1)).
				Build(),
		)
		if err == nil {
			t.Fatal("Apply() error = nil, want validation error")
		}

		var validationError ValidationError
		if !errors.As(err, &validationError) || validationError.Field != "value" {
			t.Fatalf("Apply() error = %v, want field value", err)
		}
	})

	t.Run("Named preserves validator field", func(t *testing.T) {
		_, err := Apply(
			New(setConstructorValue).
				Value(0).
				Named("outer").
				Validators(func(_ int) error {
					return ValidationError{Field: "inner", Reason: errors.New("invalid")}
				}).
				Build(),
		)
		if err == nil {
			t.Fatal("Apply() error = nil, want validation error")
		}

		var validationError ValidationError
		if !errors.As(err, &validationError) || validationError.Field != "inner" {
			t.Fatalf("Apply() error = %v, want field inner", err)
		}
	})

	t.Run("Named enriches pointer failure without mutating it", func(t *testing.T) {
		failure := &ValidationError{Reason: errors.New("invalid")}
		_, err := Apply(
			New(setConstructorValue).
				Value(0).
				Named("outer").
				Validators(func(_ int) error { return failure }).
				Build(),
		)
		if err == nil {
			t.Fatal("Apply() error = nil, want validation error")
		}

		var validationError *ValidationError
		if !errors.As(err, &validationError) || validationError.Field != "outer" {
			t.Fatalf("Apply() error = %v, want pointer failure with field outer", err)
		}
		if validationError == failure || failure.Field != "" {
			t.Fatalf("validator failure was mutated: original = %+v, returned = %+v", failure, validationError)
		}
	})

	t.Run("typed nil validation error is normalized safely", func(t *testing.T) {
		var failure *ValidationError
		_, err := Apply(
			New(setConstructorValue).
				Value(0).
				Named("value").
				Validators(func(_ int) error { return failure }).
				Build(),
		)
		if err == nil {
			t.Fatal("Apply() error = nil, want validation error")
		}

		var validationError ValidationError
		want := ValidationError{Field: "value", Reason: errors.New("<nil>")}
		if !errors.As(err, &validationError) || !sameValidationError(validationError, want) {
			t.Fatalf("Apply() error = %v, want %+v", err, want)
		}
	})

	t.Run("builder without Named leaves failure unnamed", func(t *testing.T) {
		_, err := Apply(
			New(setConstructorValue).
				Value(0).
				Validators(Min(1)).
				Build(),
		)
		if err == nil {
			t.Fatal("Apply() error = nil, want validation error")
		}

		var validationError ValidationError
		if !errors.As(err, &validationError) || validationError.Field != "" {
			t.Fatalf("Apply() error = %v, want unnamed failure", err)
		}
	})

	t.Run("custom validator", func(t *testing.T) {
		called := false
		option := New(setConstructorValue).
			Value(4).
			Validators(func(value int) error {
				called = true
				if value%2 != 0 {
					return ValidationError{Reason: errors.New("must be even")}
				}
				return nil
			}).
			Build()

		config, err := Apply(option)
		if err != nil {
			t.Fatalf("Apply() error = %v", err)
		}
		if !called || config.value != 4 {
			t.Fatalf("validator called = %t, config.value = %d", called, config.value)
		}
	})

	t.Run("validator and setter receive the same value", func(t *testing.T) {
		value := 11
		var validated *int
		option := New(func(config *constructorConfig, value *int) {
			if value != validated {
				t.Fatalf("setter value %p differs from validated value %p", value, validated)
			}
			config.ptr = value
		}).Value(&value).Validators(
			func(value *int) error {
				validated = value
				return nil
			},
		).Build()

		config, err := Apply(option)
		if err != nil {
			t.Fatalf("Apply() error = %v", err)
		}
		if config.ptr != &value {
			t.Fatalf("config.ptr = %p, want %p", config.ptr, &value)
		}
	})

	t.Run("builder reuse does not leak state", func(t *testing.T) {
		var observed []string
		baseValidator := Validator[int](func(_ int) error {
			observed = append(observed, "base")
			return nil
		})
		derivedValidator := Validator[int](func(_ int) error {
			observed = append(observed, "derived")
			return nil
		})
		provided := []Validator[int]{baseValidator}
		base := New(setConstructorValue).Validators(provided...)
		first := base.Value(2).Build()
		second := base.Validators(derivedValidator).Value(3).Build()
		provided[0] = derivedValidator

		firstConfig, err := Apply(first)
		if err != nil {
			t.Fatalf("first Apply() error = %v", err)
		}
		if firstConfig.value != 2 || !reflect.DeepEqual(observed, []string{"base"}) {
			t.Fatalf("first config = %+v, validators = %v", firstConfig, observed)
		}

		observed = nil
		secondConfig, err := Apply(second)
		if err != nil {
			t.Fatalf("second Apply() error = %v", err)
		}
		if secondConfig.value != 3 || !reflect.DeepEqual(observed, []string{"base", "derived"}) {
			t.Fatalf("second config = %+v, validators = %v", secondConfig, observed)
		}
	})

	t.Run("normalizes nested joined and wrapped errors", func(t *testing.T) {
		setterCalls := 0
		plainFailure := errors.New("plain")
		option := New(func(config *constructorConfig, value int) {
			setterCalls++
			config.value = value
		}).
			Value(5).
			Named("outer").
			Validators(func(int) error {
				return fmt.Errorf("validator context: %w", errors.Join(
					ValidationError{Value: "first-value", Reason: errors.New("first")},
					fmt.Errorf("validation context: %w", errors.Join(
						&ValidationError{
							Field:  "inner",
							Value:  "second-value",
							Reason: errors.New("second"),
						},
						fmt.Errorf("deep context: %w", ValidationError{
							Value:  "third-value",
							Reason: errors.New("third"),
						}),
					)),
					fmt.Errorf("plain context: %w", plainFailure),
				))
			}).
			Build()

		config, err := Apply(option)
		if err == nil {
			t.Fatal("Apply() error = nil, want joined validation error")
		}
		if setterCalls != 0 || config.value != 0 {
			t.Fatalf("setter calls = %d, config.value = %d, want untouched zero value", setterCalls, config.value)
		}

		want := []ValidationError{
			{Field: "outer", Value: "first-value", Reason: errors.New("first")},
			{Field: "inner", Value: "second-value", Reason: errors.New("second")},
			{Field: "outer", Value: "third-value", Reason: errors.New("third")},
		}
		if got := validationErrors(err); !sameValidationErrors(got, want) {
			t.Fatalf("Apply() errors = %+v, want %+v", got, want)
		}
		if !errors.Is(err, plainFailure) {
			t.Fatalf("Apply() error = %v, want wrapped plain failure", err)
		}
		if got := err.Error(); !strings.Contains(got, "validator context: outer: first") ||
			!strings.Contains(got, "deep context: outer: third") ||
			!strings.Contains(got, "plain context: plain") {
			t.Fatalf("Apply() error = %q, want preserved wrapper context", got)
		}
	})

	t.Run("preserves ordinary validator errors", func(t *testing.T) {
		failure := errors.New("failure")
		wrapped := fmt.Errorf("validator context: %w", failure)
		option := New(setConstructorValue).
			Value(5).
			Named("value").
			Validators(func(int) error { return wrapped }).
			Build()

		_, err := Apply(option)
		if !errors.Is(err, failure) || !errors.Is(err, wrapped) {
			t.Fatalf("Apply() error = %v, want original wrapped failure", err)
		}
	})

	t.Run("collects joined failures from every validator", func(t *testing.T) {
		var order []int
		option := New(setConstructorValue).
			Value(5).
			Validators(
				func(int) error {
					order = append(order, 1)
					return errors.Join(
						ValidationError{Reason: errors.New("first")},
						ValidationError{Reason: errors.New("second")},
					)
				},
				func(int) error {
					order = append(order, 2)
					return ValidationError{Reason: errors.New("third")}
				},
			).
			Build()

		config, err := Apply(option)
		if err == nil {
			t.Fatal("Apply() error = nil, want joined validation error")
		}
		if !reflect.DeepEqual(order, []int{1, 2}) {
			t.Fatalf("validator order = %v, want [1 2]", order)
		}
		if config.value != 0 {
			t.Fatalf("config.value = %d, want zero", config.value)
		}

		got := validationErrors(err)
		want := []ValidationError{
			{Reason: errors.New("first")},
			{Reason: errors.New("second")},
			{Reason: errors.New("third")},
		}
		if !sameValidationErrors(got, want) {
			t.Fatalf("Apply() errors = %+v, want %+v", got, want)
		}
	})
}

func validationErrors(err error) []ValidationError {
	if err == nil {
		return nil
	}

	switch err := err.(type) {
	case ValidationError:
		return []ValidationError{err}
	case *ValidationError:
		return []ValidationError{*err}
	case interface{ Unwrap() []error }:
		var result []ValidationError
		for _, child := range err.Unwrap() {
			result = append(result, validationErrors(child)...)
		}
		return result
	case interface{ Unwrap() error }:
		return validationErrors(err.Unwrap())
	default:
		return nil
	}
}

func sameValidationErrors(got, want []ValidationError) bool {
	if len(got) != len(want) {
		return false
	}

	for i := range got {
		if !sameValidationError(got[i], want[i]) {
			return false
		}
	}

	return true
}

func sameValidationError(got, want ValidationError) bool {
	return got.Field == want.Field &&
		got.Value == want.Value &&
		errorMessage(got.Reason) == errorMessage(want.Reason)
}

func errorMessage(err error) string {
	if err == nil {
		return ""
	}

	return err.Error()
}
