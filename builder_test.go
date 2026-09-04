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

type defaultConfig[V any] struct {
	value V
}

func setConstructorValue(config *constructorConfig, value int) {
	config.value = value
}

func applyDefault[V any](value, defaultValue V) (V, error) {
	option := New(func(config *defaultConfig[V], value V) {
		config.value = value
	}).Value(value).Default(defaultValue).Build()
	config, err := Apply(option)

	return config.value, err
}

func applyDefaultWhen[V any](value, defaultValue V, predicate Predicate[V]) (V, error) {
	option := New(func(config *defaultConfig[V], value V) {
		config.value = value
	}).Value(value).DefaultWhen(defaultValue, predicate).Build()
	config, err := Apply(option)

	return config.value, err
}

func TestBuilderDefault(t *testing.T) {
	t.Run("zero values use default", func(t *testing.T) {
		assertDefault := func(name string, got, want any, err error) {
			t.Helper()
			if err != nil {
				t.Fatalf("%s: Apply() error = %v", name, err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("%s: value = %#v, want %#v", name, got, want)
			}
		}

		gotInt, err := applyDefault(0, 7)
		assertDefault("int", gotInt, 7, err)

		gotString, err := applyDefault("", "default")
		assertDefault("string", gotString, "default", err)

		gotBool, err := applyDefault(false, true)
		assertDefault("bool", gotBool, true, err)

		defaultInt := 9
		gotPointer, err := applyDefault((*int)(nil), &defaultInt)
		assertDefault("pointer", gotPointer, &defaultInt, err)

		gotSlice, err := applyDefault([]string(nil), []string{"default"})
		assertDefault("slice", gotSlice, []string{"default"}, err)

		gotMap, err := applyDefault(map[string]int(nil), map[string]int{"default": 1})
		assertDefault("map", gotMap, map[string]int{"default": 1}, err)

		type count int
		gotNamed, err := applyDefault(count(0), count(3))
		assertDefault("named int", gotNamed, count(3), err)

		gotArray, err := applyDefault([2]int{}, [2]int{1, 2})
		assertDefault("array", gotArray, [2]int{1, 2}, err)

		type settings struct{ enabled bool }
		gotStruct, err := applyDefault(settings{}, settings{enabled: true})
		assertDefault("struct", gotStruct, settings{enabled: true}, err)

		var typedNil any = (*int)(nil)
		gotInterface, err := applyDefault(typedNil, any("default"))
		assertDefault("typed nil interface", gotInterface, any("default"), err)

		gotDynamicZero, err := applyDefault(any(0), any(7))
		assertDefault("dynamic zero interface", gotDynamicZero, any(7), err)

		defaultChannel := make(chan int)
		defer close(defaultChannel)
		gotChannel, err := applyDefault((chan int)(nil), defaultChannel)
		assertDefault("channel", gotChannel, defaultChannel, err)

		var nilFunction func() int
		defaultFunction := func() int { return 7 }
		gotFunction, err := applyDefault(nilFunction, defaultFunction)
		if err != nil {
			t.Fatalf("function: Apply() error = %v", err)
		}
		if gotFunction == nil || gotFunction() != 7 {
			t.Fatal("function: expected callable default")
		}
	})

	t.Run("validators and setter receive resolved default", func(t *testing.T) {
		validated := 0
		option := New(func(config *constructorConfig, value int) {
			config.value = value
		}).Value(0).Default(4).Validators(func(value int) error {
			validated = value

			return nil
		}).Build()

		config, err := Apply(option)
		if err != nil {
			t.Fatalf("Apply() error = %v", err)
		}
		if validated != 4 || config.value != 4 {
			t.Fatalf("validated = %d, value = %d, want 4, 4", validated, config.value)
		}
	})

	t.Run("non-zero values are preserved", func(t *testing.T) {
		gotInt, err := applyDefault(2, 7)
		if err != nil || gotInt != 2 {
			t.Fatalf("int value = %d, error = %v, want 2, nil", gotInt, err)
		}

		gotString, err := applyDefault("value", "default")
		if err != nil || gotString != "value" {
			t.Fatalf("string value = %q, error = %v, want value, nil", gotString, err)
		}

		gotBool, err := applyDefault(true, false)
		if err != nil || !gotBool {
			t.Fatalf("bool value = %t, error = %v, want true, nil", gotBool, err)
		}

		value := 0
		defaultValue := 9
		gotPointer, err := applyDefault(&value, &defaultValue)
		if err != nil || gotPointer != &value {
			t.Fatalf("pointer value = %p, error = %v, want %p, nil", gotPointer, err, &value)
		}

		gotSlice, err := applyDefault([]string{}, []string{"default"})
		if err != nil || gotSlice == nil || len(gotSlice) != 0 {
			t.Fatalf("slice value = %#v, error = %v, want non-nil empty slice, nil", gotSlice, err)
		}

		gotMap, err := applyDefault(map[string]int{}, map[string]int{"default": 1})
		if err != nil || gotMap == nil || len(gotMap) != 0 {
			t.Fatalf("map value = %#v, error = %v, want non-nil empty map, nil", gotMap, err)
		}

		gotSlice, err = applyDefault([]string{"value"}, []string{"default"})
		if err != nil || !reflect.DeepEqual(gotSlice, []string{"value"}) {
			t.Fatalf("populated slice = %#v, error = %v, want [value], nil", gotSlice, err)
		}

		gotMap, err = applyDefault(map[string]int{"value": 1}, map[string]int{"default": 1})
		if err != nil || !reflect.DeepEqual(gotMap, map[string]int{"value": 1}) {
			t.Fatalf("populated map = %#v, error = %v, want map[value:1], nil", gotMap, err)
		}

		type settings struct{ enabled bool }
		gotStruct, err := applyDefault(settings{enabled: true}, settings{})
		if err != nil || gotStruct != (settings{enabled: true}) {
			t.Fatalf("struct value = %+v, error = %v, want {enabled:true}, nil", gotStruct, err)
		}
	})

	t.Run("default supplies omitted value including zero", func(t *testing.T) {
		setterCalls := 0
		validatorCalls := 0
		option := New(func(config *constructorConfig, value int) {
			setterCalls++
			config.value = value
		}).Default(0).Validators(func(value int) error {
			validatorCalls++
			if value != 0 {
				return fmt.Errorf("value = %d, want 0", value)
			}

			return nil
		}).Build()

		config, err := Apply(option)
		if err != nil {
			t.Fatalf("Apply() error = %v", err)
		}
		if setterCalls != 1 || validatorCalls != 1 || config.value != 0 {
			t.Fatalf(
				"setter calls = %d, validator calls = %d, value = %d, want 1, 1, 0",
				setterCalls,
				validatorCalls,
				config.value,
			)
		}
	})

	t.Run("Value and Default are order independent and use latest calls", func(t *testing.T) {
		base := New(setConstructorValue)
		options := []Option[constructorConfig]{
			base.Value(0).Default(1).Default(2).Build(),
			base.Default(1).Default(2).Value(0).Build(),
			base.Default(1).Value(0).Value(3).Build(),
		}
		wants := []int{2, 2, 3}

		for i, option := range options {
			config, err := Apply(option)
			if err != nil {
				t.Fatalf("option %d: Apply() error = %v", i, err)
			}
			if config.value != wants[i] {
				t.Fatalf("option %d: value = %d, want %d", i, config.value, wants[i])
			}
		}

		_, err := Apply(base.Build())
		if err == nil || err.Error() != "option value was not provided" {
			t.Fatalf("base Apply() error = %v, want missing-value error", err)
		}
	})

	t.Run("validators inspect resolved default and suppress setter on failure", func(t *testing.T) {
		initial := constructorConfig{value: 9}
		setterCalls := 0
		validated := -1
		failure := errors.New("invalid default")
		option := New(func(config *constructorConfig, value int) {
			setterCalls++
			config.value = value
		}).Value(0).Default(4).Named("value").Validators(func(value int) error {
			validated = value

			return failure
		}).Build()

		config, err := ApplyTo(initial, option)
		if err == nil {
			t.Fatal("ApplyTo() error = nil, want validation error")
		}
		if setterCalls != 0 || validated != 4 || config != initial {
			t.Fatalf(
				"setter calls = %d, validated = %d, config = %+v, want 0, 4, %+v",
				setterCalls,
				validated,
				config,
				initial,
			)
		}

		var validationError ValidationError
		want := ValidationError{Field: "value", Value: "4", Reason: failure}
		if !errors.As(err, &validationError) || !sameValidationError(validationError, want) {
			t.Fatalf("ApplyTo() error = %v, want %+v", err, want)
		}
	})
}

func TestBuilderDefaultWhen(t *testing.T) {
	t.Run("predicate controls fallback", func(t *testing.T) {
		useDefault := Predicate[int](func(value int) bool { return value < 0 })

		got, err := applyDefaultWhen(-1, 7, useDefault)
		if err != nil || got != 7 {
			t.Fatalf("matching value = %d, error = %v, want 7, nil", got, err)
		}

		got, err = applyDefaultWhen(2, 7, useDefault)
		if err != nil || got != 2 {
			t.Fatalf("non-matching value = %d, error = %v, want 2, nil", got, err)
		}

		got, err = applyDefaultWhen(0, 7, useDefault)
		if err != nil || got != 0 {
			t.Fatalf("non-matching zero value = %d, error = %v, want 0, nil", got, err)
		}
	})

	t.Run("supports domain-specific empty values", func(t *testing.T) {
		blank := Predicate[string](func(value string) bool {
			return strings.TrimSpace(value) == ""
		})

		got, err := applyDefaultWhen(" \t", "fallback", blank)
		if err != nil || got != "fallback" {
			t.Fatalf("blank value = %q, error = %v, want fallback, nil", got, err)
		}

		got, err = applyDefaultWhen("value", "fallback", blank)
		if err != nil || got != "value" {
			t.Fatalf("non-blank value = %q, error = %v, want value, nil", got, err)
		}
	})

	t.Run("supports collection helper predicates", func(t *testing.T) {
		fallbackSlice := []string{"fallback"}
		gotSlice, err := applyDefaultWhen([]string{}, fallbackSlice, EmptySlice[[]string]())
		if err != nil || !reflect.DeepEqual(gotSlice, fallbackSlice) {
			t.Fatalf("empty slice = %#v, error = %v, want %#v, nil", gotSlice, err, fallbackSlice)
		}

		gotSlice, err = applyDefaultWhen([]string{}, fallbackSlice, NilSlice[[]string]())
		if err != nil || gotSlice == nil || len(gotSlice) != 0 {
			t.Fatalf("non-nil slice = %#v, error = %v, want non-nil empty slice, nil", gotSlice, err)
		}

		fallbackMap := map[string]int{"fallback": 1}
		gotMap, err := applyDefaultWhen(map[string]int{}, fallbackMap, EmptyMap[map[string]int]())
		if err != nil || !reflect.DeepEqual(gotMap, fallbackMap) {
			t.Fatalf("empty map = %#v, error = %v, want %#v, nil", gotMap, err, fallbackMap)
		}

		gotMap, err = applyDefaultWhen(map[string]int{}, fallbackMap, NilMap[map[string]int]())
		if err != nil || gotMap == nil || len(gotMap) != 0 {
			t.Fatalf("non-nil map = %#v, error = %v, want non-nil empty map, nil", gotMap, err)
		}
	})

	t.Run("Value and DefaultWhen are order independent", func(t *testing.T) {
		predicate := Predicate[int](func(value int) bool { return value == 0 })
		base := New(setConstructorValue)
		options := []Option[constructorConfig]{
			base.Value(0).DefaultWhen(4, predicate).Build(),
			base.DefaultWhen(4, predicate).Value(0).Build(),
		}

		for i, option := range options {
			config, err := Apply(option)
			if err != nil {
				t.Fatalf("option %d: Apply() error = %v", i, err)
			}
			if config.value != 4 {
				t.Fatalf("option %d: value = %d, want 4", i, config.value)
			}
		}
	})

	t.Run("missing value uses default without evaluating predicate", func(t *testing.T) {
		predicateCalls := 0
		option := New(setConstructorValue).DefaultWhen(4, func(int) bool {
			predicateCalls++

			return false
		}).Build()
		if predicateCalls != 0 {
			t.Fatalf("predicate calls after Build = %d, want 0", predicateCalls)
		}

		config, err := Apply(option)
		if err != nil {
			t.Fatalf("Apply() error = %v", err)
		}
		if predicateCalls != 0 || config.value != 4 {
			t.Fatalf("predicate calls = %d, value = %d, want 0, 4", predicateCalls, config.value)
		}
	})

	t.Run("nil predicate preserves supplied value and defaults missing value", func(t *testing.T) {
		withValue, err := Apply(
			New(setConstructorValue).Value(0).DefaultWhen(4, nil).Build(),
		)
		if err != nil || withValue.value != 0 {
			t.Fatalf("supplied value = %d, error = %v, want 0, nil", withValue.value, err)
		}

		withoutValue, err := Apply(
			New(setConstructorValue).DefaultWhen(4, nil).Build(),
		)
		if err != nil || withoutValue.value != 4 {
			t.Fatalf("missing value = %d, error = %v, want 4, nil", withoutValue.value, err)
		}
	})

	t.Run("latest fallback policy wins and builder reuse remains isolated", func(t *testing.T) {
		always := Predicate[int](func(int) bool { return true })
		never := Predicate[int](func(int) bool { return false })
		base := New(setConstructorValue).Value(1)
		options := []Option[constructorConfig]{
			base.DefaultWhen(2, never).DefaultWhen(3, always).Build(),
			base.DefaultWhen(2, always).Default(3).Build(),
			base.Default(2).DefaultWhen(3, always).Build(),
			base.Build(),
		}
		wants := []int{3, 1, 3, 1}

		for i, option := range options {
			config, err := Apply(option)
			if err != nil {
				t.Fatalf("option %d: Apply() error = %v", i, err)
			}
			if config.value != wants[i] {
				t.Fatalf("option %d: value = %d, want %d", i, config.value, wants[i])
			}
		}
	})

	t.Run("predicate is evaluated once during Build", func(t *testing.T) {
		predicateCalls := 0
		builder := New(setConstructorValue).Value(0).DefaultWhen(4, func(int) bool {
			predicateCalls++

			return true
		})
		option := builder.Build()
		if predicateCalls != 1 {
			t.Fatalf("predicate calls after Build = %d, want 1", predicateCalls)
		}

		for range 2 {
			config, err := Apply(option)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			if config.value != 4 {
				t.Fatalf("config.value = %d, want 4", config.value)
			}
		}
		if predicateCalls != 1 {
			t.Fatalf("predicate calls after Apply = %d, want 1", predicateCalls)
		}
	})

	t.Run("validators inspect resolved value and suppress setter on failure", func(t *testing.T) {
		initial := constructorConfig{value: 9}
		setterCalls := 0
		validated := ""
		failure := errors.New("invalid default")
		option := New(func(config *constructorConfig, value string) {
			setterCalls++
			config.value = len(value)
		}).Value(" ").DefaultWhen("fallback", func(value string) bool {
			return strings.TrimSpace(value) == ""
		}).Named("value").Validators(func(value string) error {
			validated = value

			return failure
		}).Build()

		config, err := ApplyTo(initial, option)
		if err == nil {
			t.Fatal("ApplyTo() error = nil, want validation error")
		}
		if setterCalls != 0 || validated != "fallback" || config != initial {
			t.Fatalf(
				"setter calls = %d, validated = %q, config = %+v, want 0, fallback, %+v",
				setterCalls,
				validated,
				config,
				initial,
			)
		}

		var validationError ValidationError
		want := ValidationError{Field: "value", Value: "fallback", Reason: failure}
		if !errors.As(err, &validationError) || !sameValidationError(validationError, want) {
			t.Fatalf("ApplyTo() error = %v, want %+v", err, want)
		}
	})
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
