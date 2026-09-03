package options_test

import (
	"errors"
	"fmt"
	"time"

	options "github.com/ziflex/go-options"
)

type exampleConfig struct {
	timeout time.Duration
	workers int
}

func ExampleNew() {
	option := options.New(func(config *exampleConfig, value int) {
		config.workers = value
	}).
		Value(4).
		Validators(options.Min(1), options.Max(32)).
		Build()
	config, err := options.Apply(option)
	fmt.Println(config.workers, err)

	// Output:
	// 4 <nil>
}

func ExampleBuilder() {
	option := options.New(
		func(config *exampleConfig, value time.Duration) {
			config.timeout = value
		},
	).
		Value(5 * time.Second).
		Named("timeout").
		Validators(options.Min(time.Second)).
		Build()
	config, err := options.Apply(option)
	fmt.Println(config.timeout, err)

	// Output:
	// 5s <nil>
}

func ExampleBuilder_Default() {
	option := options.New(func(config *exampleConfig, value int) {
		config.workers = value
	}).Value(0).Default(4).Build()
	config, err := options.Apply(option)
	fmt.Println(config.workers, err)

	// Output:
	// 4 <nil>
}

func ExampleBuilder_Named() {
	option := options.New(
		func(config *exampleConfig, value int) {
			config.workers = value
		},
	).
		Value(0).
		Named("workers").
		Validators(func(value int) error {
			if value < 1 {
				return errors.New("must be greater than or equal to 1")
			}

			return nil
		}).
		Build()

	_, err := options.Apply(option)
	fmt.Println(err)

	// Output:
	// workers: must be greater than or equal to 1: value=0
}

func ExampleValidator() {
	even := options.Validator[int](func(value int) error {
		if value%2 != 0 {
			return options.ValidationError{Reason: errors.New("must be even")}
		}

		return nil
	})
	option := options.New(
		func(config *exampleConfig, value int) {
			config.workers = value
		},
	).
		Value(4).
		Validators(even).
		Build()

	config, err := options.Apply(option)
	fmt.Println(config.workers, err)

	// Output:
	// 4 <nil>
}

func ExampleCheck() {
	even := options.Check(func(value int) error {
		if value%2 != 0 {
			return errors.New("must be even")
		}

		return nil
	})

	fmt.Println(even(4))
	fmt.Println(even(3))

	// Output:
	// <nil>
	// must be even
}

func ExampleNotNil() {
	type resource struct{}
	type config struct {
		resource *resource
	}

	option := options.New(
		func(config *config, value *resource) {
			config.resource = value
		},
	).
		Value((*resource)(nil)).
		Validators(options.NotNil[*resource]()).
		Build()

	_, err := options.Apply(option)
	fmt.Println(err)

	// Output:
	// must not be nil: value=<nil>
}

func ExampleNotNilPtr() {
	type resource struct{}
	type config struct {
		resource *resource
	}

	option := options.New(
		func(config *config, value *resource) {
			config.resource = value
		},
	).
		Value((*resource)(nil)).
		Validators(options.NotNilPtr[resource]()).
		Build()

	_, err := options.Apply(option)
	fmt.Println(err)

	// Output:
	// cannot be nil: value=<nil>
}

func ExampleApplyTo() {
	defaults := exampleConfig{timeout: 30 * time.Second, workers: 2}
	option := options.New(func(config *exampleConfig, value int) {
		config.workers = value
	}).Value(4).Build()
	config, err := options.ApplyTo(defaults, option)
	fmt.Println(config.timeout, config.workers, err)

	// Output:
	// 30s 4 <nil>
}
