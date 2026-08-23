package options_test

import (
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

func TestCustomValidatorContract(t *testing.T) {
	var validator options.Validator[int] = func(value int) error {
		if value < 0 {
			return options.ValidationError{Reason: "must not be negative"}
		}

		return nil
	}

	if err := validator(1); err != nil {
		t.Fatalf("validator(1) error = %v", err)
	}
}
