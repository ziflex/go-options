package options

import (
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
				Reason: "invalid timeout",
				Value:  "-1",
			},
			want: "Timeout: invalid timeout: value=-1",
		},
		{
			name: "no field",
			err: ValidationError{
				Reason: "something went wrong",
				Value:  "foo",
			},
			want: "something went wrong: value=foo",
		},
		{
			name: "no value",
			err: ValidationError{
				Field:  "Name",
				Reason: "cannot be empty",
			},
			want: "Name: cannot be empty",
		},
		{
			name: "only reason",
			err: ValidationError{
				Reason: "fatal error",
			},
			want: "fatal error",
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
