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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("ValidationError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
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
