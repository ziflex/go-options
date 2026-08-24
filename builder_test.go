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
		firstFailure := errors.New("first")
		thirdFailure := errors.New("third")
		option := New(setConstructorValue).
			Value(5).
			Validators(func(_ int) error {
				order = append(order, 1)
				return firstFailure
			}).
			Validators(
				nil,
				func(_ int) error {
					order = append(order, 2)
					return nil
				},
				func(_ int) error {
					order = append(order, 3)
					return thirdFailure
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
		if !errors.Is(err, firstFailure) || !errors.Is(err, thirdFailure) {
			t.Fatalf("Apply() error = %v, want both validator failures", err)
		}
	})

	t.Run("Named wraps ordinary validator error", func(t *testing.T) {
		failure := errors.New("invalid")
		_, err := Apply(
			New(setConstructorValue).
				Value(0).
				Named("old").
				Named("value").
				Validators(func(int) error { return failure }).
				Build(),
		)
		if err == nil {
			t.Fatal("Apply() error = nil, want validation error")
		}

		var validationError ValidationError
		want := ValidationError{Field: "value", Value: "0", Reason: failure}
		if !errors.As(err, &validationError) || !sameValidationError(validationError, want) {
			t.Fatalf("Apply() error = %v, want %+v", err, want)
		}
		if validationError.Reason != failure || !errors.Is(err, failure) {
			t.Fatalf("Apply() error = %v, want original validator failure", err)
		}
	})

	t.Run("Named preserves explicit validator field", func(t *testing.T) {
		reason := errors.New("invalid")
		failure := &ValidationError{
			Field:  "inner",
			Value:  "inner-value",
			Reason: reason,
		}
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

		var outer ValidationError
		wantOuter := ValidationError{Field: "outer", Value: "0", Reason: failure}
		if !errors.As(err, &outer) || !sameValidationError(outer, wantOuter) {
			t.Fatalf("Apply() error = %v, want outer failure %+v", err, wantOuter)
		}

		var inner *ValidationError
		if !errors.As(outer.Reason, &inner) || inner != failure {
			t.Fatalf("outer reason = %v, want original nested failure", outer.Reason)
		}
		wantInner := ValidationError{Field: "inner", Value: "inner-value", Reason: reason}
		if !sameValidationError(*failure, wantInner) {
			t.Fatalf("validator failure = %+v, want unchanged %+v", failure, wantInner)
		}
	})

	t.Run("Named enriches direct fieldless pointer without mutation", func(t *testing.T) {
		reason := errors.New("invalid")
		failure := &ValidationError{Value: "inner-value", Reason: reason}
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

		var normalized *ValidationError
		want := ValidationError{Field: "outer", Value: "inner-value", Reason: reason}
		if !errors.As(err, &normalized) || normalized == nil || !sameValidationError(*normalized, want) {
			t.Fatalf("Apply() error = %v, want normalized failure %+v", err, want)
		}
		if normalized == failure || normalized.Reason != reason {
			t.Fatalf("Apply() error = %v, want a copied flat validation failure", err)
		}
		if failure.Field != "" || failure.Value != "inner-value" || failure.Reason != reason {
			t.Fatalf("validator failure was mutated: %+v", failure)
		}
	})

	t.Run("Named fills missing direct validation value", func(t *testing.T) {
		reason := errors.New("invalid")
		_, err := Apply(
			New(setConstructorValue).
				Value(0).
				Named("value").
				Validators(func(_ int) error {
					return ValidationError{Reason: reason}
				}).
				Build(),
		)
		if err == nil {
			t.Fatal("Apply() error = nil, want validation error")
		}

		var validationError ValidationError
		want := ValidationError{Field: "value", Value: "0", Reason: reason}
		if !errors.As(err, &validationError) || !sameValidationError(validationError, want) {
			t.Fatalf("Apply() error = %v, want %+v", err, want)
		}
		if validationError.Reason != reason {
			t.Fatalf("Apply() error = %v, want original reason", err)
		}
	})

	t.Run("named built-in validator produces one flat failure", func(t *testing.T) {
		_, err := Apply(
			New(setConstructorValue).
				Value(-1).
				Named("value").
				Validators(NonNegative[int]()).
				Build(),
		)
		if err == nil {
			t.Fatal("Apply() error = nil, want validation error")
		}

		var validationError ValidationError
		want := ValidationError{Field: "value", Value: "-1", Reason: errors.New("must be non-negative")}
		if !errors.As(err, &validationError) || !sameValidationError(validationError, want) {
			t.Fatalf("Apply() error = %v, want %+v", err, want)
		}

		var nested ValidationError
		if errors.As(validationError.Reason, &nested) {
			t.Fatalf("validation reason = %+v, want a non-validation leaf error", validationError.Reason)
		}
		if count := strings.Count(err.Error(), "value=-1"); count != 1 {
			t.Fatalf("Apply() error = %q, want one rendered invalid value, got %d", err, count)
		}
	})

	t.Run("builder without Named wraps failure without a field", func(t *testing.T) {
		failure := errors.New("invalid")
		_, err := Apply(
			New(setConstructorValue).
				Value(0).
				Validators(func(_ int) error { return failure }).
				Build(),
		)
		if err == nil {
			t.Fatal("Apply() error = nil, want validation error")
		}

		var validationError ValidationError
		want := ValidationError{Value: "0", Reason: failure}
		if !errors.As(err, &validationError) || !sameValidationError(validationError, want) {
			t.Fatalf("Apply() error = %v, want %+v", err, want)
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

	t.Run("preserves nested joined and wrapped errors", func(t *testing.T) {
		setterCalls := 0
		firstFailure := errors.New("first")
		secondFailure := errors.New("second")
		inner := &ValidationError{
			Field:  "inner",
			Value:  "inner-value",
			Reason: firstFailure,
		}
		joined := errors.Join(inner, secondFailure)
		wrapped := fmt.Errorf("validator context: %w", joined)
		option := New(func(config *constructorConfig, value int) {
			setterCalls++
			config.value = value
		}).
			Value(5).
			Named("outer").
			Validators(func(int) error { return wrapped }).
			Build()

		config, err := Apply(option)
		if err == nil {
			t.Fatal("Apply() error = nil, want joined validation error")
		}
		if setterCalls != 0 || config.value != 0 {
			t.Fatalf("setter calls = %d, config.value = %d, want untouched zero value", setterCalls, config.value)
		}

		var outer ValidationError
		wantOuter := ValidationError{Field: "outer", Value: "5", Reason: wrapped}
		if !errors.As(err, &outer) || !sameValidationError(outer, wantOuter) {
			t.Fatalf("Apply() error = %v, want outer failure %+v", err, wantOuter)
		}
		if outer.Reason != wrapped || !errors.Is(err, wrapped) || !errors.Is(err, joined) {
			t.Fatalf("Apply() error = %v, want original wrapped and joined errors", err)
		}
		if !errors.Is(err, firstFailure) || !errors.Is(err, secondFailure) {
			t.Fatalf("Apply() error = %v, want both nested failures", err)
		}

		var gotInner *ValidationError
		if !errors.As(outer.Reason, &gotInner) || gotInner != inner {
			t.Fatalf("outer reason = %v, want original nested validation error", outer.Reason)
		}
		wantInner := ValidationError{Field: "inner", Value: "inner-value", Reason: firstFailure}
		if !sameValidationError(*inner, wantInner) {
			t.Fatalf("nested validation error = %+v, want unchanged %+v", inner, wantInner)
		}
	})

	t.Run("wraps ordinary wrapped validator error", func(t *testing.T) {
		failure := errors.New("failure")
		wrapped := fmt.Errorf("validator context: %w", failure)
		option := New(setConstructorValue).
			Value(5).
			Named("value").
			Validators(func(int) error { return wrapped }).
			Build()

		_, err := Apply(option)
		if err == nil {
			t.Fatal("Apply() error = nil, want validation error")
		}

		var outer ValidationError
		want := ValidationError{Field: "value", Value: "5", Reason: wrapped}
		if !errors.As(err, &outer) || !sameValidationError(outer, want) || outer.Reason != wrapped {
			t.Fatalf("Apply() error = %v, want %+v", err, want)
		}
		if !errors.Is(err, failure) || !errors.Is(err, wrapped) {
			t.Fatalf("Apply() error = %v, want original wrapped failure", err)
		}
	})

	t.Run("collects joined failures from every validator", func(t *testing.T) {
		var order []int
		firstFailure := errors.New("first")
		secondFailure := errors.New("second")
		thirdFailure := errors.New("third")
		option := New(setConstructorValue).
			Value(5).
			Named("value").
			Validators(
				func(int) error {
					order = append(order, 1)
					return errors.Join(firstFailure, secondFailure)
				},
				func(int) error {
					order = append(order, 2)
					return thirdFailure
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
		if !errors.Is(err, firstFailure) || !errors.Is(err, secondFailure) || !errors.Is(err, thirdFailure) {
			t.Fatalf("Apply() error = %v, want all validator failures", err)
		}

		var outer ValidationError
		if !errors.As(err, &outer) || outer.Field != "value" || outer.Value != "5" {
			t.Fatalf("Apply() error = %v, want option-level validation context", err)
		}
	})
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
