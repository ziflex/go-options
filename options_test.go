package options

import (
	"errors"
	"testing"
)

type Config struct {
	Name    string
	Timeout int
	Debug   bool
}

func WithName(name string) Option[Config] {
	return func(c *Config, _ Report) {
		c.Name = name
	}
}

func WithTimeout(timeout int) Option[Config] {
	return func(c *Config, report Report) {
		if timeout < 0 {
			report(ValidationError{
				Field:  "Timeout",
				Reason: "timeout cannot be negative",
				Value:  "invalid",
			})
			return
		}
		c.Timeout = timeout
	}
}

func TestApply(t *testing.T) {
	t.Run("empty options", func(t *testing.T) {
		cfg, err := Apply[Config]()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if cfg.Name != "" || cfg.Timeout != 0 || cfg.Debug != false {
			t.Errorf("expected zero values, got %+v", cfg)
		}
	})

	t.Run("with options", func(t *testing.T) {
		cfg, err := Apply(WithName("test"), WithTimeout(10))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if cfg.Name != "test" || cfg.Timeout != 10 {
			t.Errorf("expected updated values, got %+v", cfg)
		}
	})

	t.Run("with errors", func(t *testing.T) {
		cfg, err := Apply(WithTimeout(-1))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var valErr ValidationError
		if !errors.As(err, &valErr) {
			t.Errorf("expected ValidationError, got %T", err)
		}
		if cfg.Timeout != 0 {
			t.Errorf("expected timeout to be 0, got %d", cfg.Timeout)
		}
	})

	t.Run("with multiple errors", func(t *testing.T) {
		_, err := Apply(WithTimeout(-1), WithTimeout(-2))
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		var errs interface{ Unwrap() []error }
		if !errors.As(err, &errs) {
			t.Errorf("expected joined error, got %T", err)
		} else if len(errs.Unwrap()) != 2 {
			t.Errorf("expected 2 errors, got %d", len(errs.Unwrap()))
		}
	})

	t.Run("with nil option", func(t *testing.T) {
		cfg, err := Apply[Config](nil, WithName("test"))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if cfg.Name != "test" {
			t.Errorf("expected Name to be test, got %s", cfg.Name)
		}
	})
}

func TestApplyTo(t *testing.T) {
	initial := Config{Name: "initial", Timeout: 5}

	t.Run("no options", func(t *testing.T) {
		cfg, err := ApplyTo(initial)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if cfg != initial {
			t.Errorf("expected %+v, got %+v", initial, cfg)
		}
	})

	t.Run("override values", func(t *testing.T) {
		cfg, err := ApplyTo(initial, WithName("override"))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if cfg.Name != "override" || cfg.Timeout != 5 {
			t.Errorf("expected overridden Name and initial Timeout, got %+v", cfg)
		}
	})

	t.Run("invalid option preserves defaults and later options continue", func(t *testing.T) {
		cfg, err := ApplyTo(initial, WithTimeout(-1), WithName("later"))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if cfg.Name != "later" || cfg.Timeout != initial.Timeout {
			t.Errorf("expected later Name and initial Timeout, got %+v", cfg)
		}
	})
}

func TestApplyWithValues(t *testing.T) {
	initial := Config{Name: "initial", Timeout: 5}
	options := []Option[Config]{WithName("override"), WithTimeout(-1)}

	want, wantErr := ApplyTo(initial, options...)
	got, gotErr := ApplyWithValues(initial, options...)

	if got != want {
		t.Fatalf("ApplyWithValues() = %+v, ApplyTo() = %+v", got, want)
	}
	if (gotErr == nil) != (wantErr == nil) || (gotErr != nil && gotErr.Error() != wantErr.Error()) {
		t.Fatalf("ApplyWithValues() error = %v, ApplyTo() error = %v", gotErr, wantErr)
	}
}
