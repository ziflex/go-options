package options_test

import (
	"errors"
	"testing"

	options "github.com/ziflex/go-options"
)

func TestCollectionValidatorTypeInference(t *testing.T) {
	type names []string
	type labels map[string]int

	var sliceValidator options.Validator[names] = options.SliceMinLen[names](1)
	var mapValidator options.Validator[labels] = options.MapMaxLen[labels](2)

	if sliceValidator == nil || mapValidator == nil {
		t.Fatal("expected collection validators")
	}
}

func TestNumericSignValidatorTypes(t *testing.T) {
	type count int64
	type ratio float32
	type size uint

	var positive options.Validator[count] = options.Positive[count]()
	var nonNegative options.Validator[ratio] = options.NonNegative[ratio]()
	var negative options.Validator[count] = options.Negative[count]()
	var nonPositive options.Validator[size] = options.NonPositive[size]()

	if positive == nil || nonNegative == nil || negative == nil || nonPositive == nil {
		t.Fatal("expected numeric sign validators")
	}
}

func TestCheckTypeInference(t *testing.T) {
	type count int

	var validator options.Validator[count] = options.Check(func(value count) error {
		if value < 0 {
			return errors.New("must not be negative")
		}

		return nil
	})
	if err := validator(1); err != nil {
		t.Fatalf("validator(1) error = %v", err)
	}
}

func TestCustomValidatorContract(t *testing.T) {
	var validator options.Validator[int] = func(value int) error {
		if value < 0 {
			return options.ValidationError{Reason: errors.New("must not be negative")}
		}

		return nil
	}

	if err := validator(1); err != nil {
		t.Fatalf("validator(1) error = %v", err)
	}
}

func TestCustomOptionContract(t *testing.T) {
	type config struct {
		value int
	}

	var option options.Option[config] = func(config *config) error {
		config.value = 1

		return nil
	}

	got, err := options.Apply(option)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if got.value != 1 {
		t.Fatalf("config.value = %d, want 1", got.value)
	}
}
