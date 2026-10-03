package options

import (
	"errors"
	"fmt"
	"testing"
)

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  ValidationError
		want string
	}{
		{
			name: "full error",
			err: ValidationError{
				Field:  "Timeout",
				Reason: errors.New("invalid timeout"),
				Value:  "-1",
			},
			want: "Timeout: invalid timeout: value=-1",
		},
		{
			name: "no field",
			err: ValidationError{
				Reason: errors.New("something went wrong"),
				Value:  "foo",
			},
			want: "something went wrong: value=foo",
		},
		{
			name: "no value",
			err: ValidationError{
				Field:  "Name",
				Reason: errors.New("cannot be empty"),
			},
			want: "Name: cannot be empty",
		},
		{
			name: "only reason",
			err: ValidationError{
				Reason: errors.New("fatal error"),
			},
			want: "fatal error",
		},
		{
			name: "nil reason",
			err:  ValidationError{},
			want: "",
		},
		{
			name: "nil reason with context",
			err: ValidationError{
				Field: "Name",
				Value: `""`,
			},
			want: `Name: : value=""`,
		},
		{
			name: "omitted populated value",
			err: ValidationError{
				Field: "Name", Value: "hidden", OmitValue: true,
				Reason: errors.New("invalid"),
			},
			want: "Name: invalid",
		},
		{
			name: "omission preserves child details",
			err: ValidationError{
				Field: `key["visible"]`, Value: "hidden", OmitValue: true,
				Reason: ValidationError{Field: "child", Value: "visible", Reason: errors.New("invalid")},
			},
			want: `key["visible"]: child: invalid: value=visible`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("ValidationError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToValidationError(t *testing.T) {
	reason := &validationReason{message: "invalid"}
	for _, field := range []string{"", "child"} {
		for _, value := range []string{"", "explicit"} {
			for _, omit := range []bool{false, true} {
				for _, pointer := range []bool{false, true} {
					name := fmt.Sprintf("field=%q/value=%q/omit=%t/pointer=%t", field, value, omit, pointer)
					t.Run(name, func(t *testing.T) {
						original := ValidationError{Field: field, Value: value, Reason: reason, OmitValue: omit}
						owned := original
						var child error = original
						if pointer {
							child = &owned
						}

						got := ToValidationError("option", "fallback", child)
						want := original
						want.Field = "option"
						if field != "" {
							want = ValidationError{Field: "option", Reason: child, OmitValue: omit}
						}
						if !omit && (field != "" || value == "") {
							want.Value = "fallback"
						}

						var normalized ValidationError
						if pointer && field == "" {
							copy, ok := got.(*ValidationError)
							if !ok || copy == &owned {
								t.Fatal("expected a copied pointer validation error")
							}
							normalized = *copy
						} else {
							var ok bool
							normalized, ok = got.(ValidationError)
							if !ok {
								t.Fatalf("normalized type = %T, want ValidationError", got)
							}
						}
						if !sameValidationError(normalized, want) || normalized.Reason != want.Reason {
							t.Fatalf("normalized = %+v, want %+v", normalized, want)
						}
						if owned != original {
							t.Fatalf("caller-owned error changed: %+v", owned)
						}
						var cause *validationReason
						if !errors.Is(got, reason) || !errors.As(got, &cause) || cause != reason {
							t.Fatal("original cause is not discoverable")
						}
					})
				}
			}
		}
	}

	t.Run("opaque and nil errors preserve wrapping", func(t *testing.T) {
		omitted := &ValidationError{OmitValue: true, Reason: reason}
		var nilPointer *ValidationError
		for _, child := range []error{reason, fmt.Errorf("context: %w", omitted), errors.Join(omitted, reason), nilPointer, nil} {
			got := ToValidationError("option", "fallback", child)
			outer, ok := got.(ValidationError)
			if !ok || outer.Field != "option" || outer.Value != "fallback" || outer.OmitValue || outer.Reason != child {
				t.Fatalf("normalization did not preserve wrapping for %T", child)
			}
		}
	})
}

func TestValidationError_Unwrap(t *testing.T) {
	reason := &validationReason{message: "invalid"}
	err := ValidationError{Reason: reason}

	if got := err.Unwrap(); got != reason {
		t.Fatalf("ValidationError.Unwrap() = %v, want reason", got)
	}
	chain := errors.Join(errors.New("other"), fmt.Errorf("context: %w", err))
	if !errors.Is(chain, reason) {
		t.Fatalf("errors.Is(%v, reason) = false", chain)
	}

	var target *validationReason
	if !errors.As(chain, &target) || target != reason {
		t.Fatalf("errors.As(%v) = %v, want reason", chain, target)
	}
}

type validationReason struct {
	message string
}

func (e *validationReason) Error() string {
	return e.message
}
