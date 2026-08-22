package options

import (
	"errors"
	"reflect"
	"testing"
)

type constructorConfig struct {
	value int
	ptr   *int
}

func TestWith(t *testing.T) {
	t.Run("without validators", func(t *testing.T) {
		calls := 0
		option := With(7, func(config *constructorConfig, value int) {
			calls++
			config.value = value
		})

		config, err := Apply(option)
		if err != nil {
			t.Fatalf("Apply() error = %v", err)
		}
		if calls != 1 || config.value != 7 {
			t.Fatalf("setter calls = %d, config.value = %d", calls, config.value)
		}
	})

	t.Run("passing validator", func(t *testing.T) {
		config, err := Apply(With(
			7,
			func(config *constructorConfig, value int) { config.value = value },
			Min(1),
		))
		if err != nil {
			t.Fatalf("Apply() error = %v", err)
		}
		if config.value != 7 {
			t.Fatalf("config.value = %d, want 7", config.value)
		}
	})

	t.Run("invalid value is not assigned", func(t *testing.T) {
		initial := constructorConfig{value: 9}
		calls := 0
		config, err := ApplyTo(initial, With(
			0,
			func(config *constructorConfig, value int) {
				calls++
				config.value = value
			},
			Min(1),
		))
		if err == nil {
			t.Fatal("ApplyTo() error = nil, want validation error")
		}
		if calls != 0 || config != initial {
			t.Fatalf("setter calls = %d, config = %+v, want %+v", calls, config, initial)
		}
	})

	t.Run("all validators run in order", func(t *testing.T) {
		var order []int
		option := With(
			5,
			func(config *constructorConfig, value int) { config.value = value },
			Check(func(_ int, report Report) {
				order = append(order, 1)
				report(ValidationError{Reason: "first"})
			}),
			nil,
			Check(func(_ int, report Report) {
				order = append(order, 2)
				report(ValidationError{Reason: "second"})
			}),
		)

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

		var joined interface{ Unwrap() []error }
		if !errors.As(err, &joined) || len(joined.Unwrap()) != 2 {
			t.Fatalf("Apply() error = %v, want two joined errors", err)
		}
	})

	t.Run("validator and setter receive the same value", func(t *testing.T) {
		value := 11
		var validated *int
		config, err := Apply(With(
			&value,
			func(config *constructorConfig, value *int) {
				if value != validated {
					t.Fatalf("setter value %p differs from validated value %p", value, validated)
				}
				config.ptr = value
			},
			Check(func(value *int, _ Report) { validated = value }),
		))
		if err != nil {
			t.Fatalf("Apply() error = %v", err)
		}
		if config.ptr != &value {
			t.Fatalf("config.ptr = %p, want %p", config.ptr, &value)
		}
	})
}

func TestNew(t *testing.T) {
	setter := func(config *constructorConfig, value int) { config.value = value }
	withValue := New(setter, Min(1))

	t.Run("reuses constructor for multiple values", func(t *testing.T) {
		first, err := Apply(withValue(2))
		if err != nil || first.value != 2 {
			t.Fatalf("first Apply() = %+v, %v", first, err)
		}

		second, err := Apply(withValue(3), withValue(4))
		if err != nil || second.value != 4 {
			t.Fatalf("second Apply() = %+v, %v", second, err)
		}
	})

	t.Run("invalid value is not assigned", func(t *testing.T) {
		initial := constructorConfig{value: 8}
		config, err := ApplyTo(initial, withValue(0))
		if err == nil {
			t.Fatal("ApplyTo() error = nil, want validation error")
		}
		if config != initial {
			t.Fatalf("config = %+v, want %+v", config, initial)
		}
	})

	t.Run("later valid option still applies", func(t *testing.T) {
		initial := constructorConfig{value: 8}
		config, err := ApplyTo(initial, withValue(0), withValue(4))
		if err == nil {
			t.Fatal("ApplyTo() error = nil, want validation error")
		}
		if config.value != 4 {
			t.Fatalf("config.value = %d, want 4", config.value)
		}
	})

	t.Run("matches With", func(t *testing.T) {
		fromNew, newErr := Apply(withValue(0))
		fromWith, withErr := Apply(With(0, setter, Min(1)))
		if fromNew != fromWith {
			t.Fatalf("New result = %+v, With result = %+v", fromNew, fromWith)
		}
		if (newErr == nil) != (withErr == nil) || (newErr != nil && newErr.Error() != withErr.Error()) {
			t.Fatalf("New error = %v, With error = %v", newErr, withErr)
		}
	})
}
