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
