package options_test

import (
	"fmt"
	"time"

	options "github.com/ziflex/go-options"
)

type exampleConfig struct {
	timeout time.Duration
	workers int
}

var withTimeout = options.New(
	func(config *exampleConfig, value time.Duration) {
		config.timeout = value
	},
	options.Min(time.Second),
)

func ExampleNew() {
	config, err := options.Apply(withTimeout(5 * time.Second))
	fmt.Println(config.timeout, err)

	// Output:
	// 5s <nil>
}

func withWorkers(value int) options.Option[exampleConfig] {
	return options.With(
		value,
		func(config *exampleConfig, value int) {
			config.workers = value
		},
		options.Min(1),
		options.Max(32),
	)
}

func ExampleWith() {
	config, err := options.Apply(withWorkers(4))
	fmt.Println(config.workers, err)

	// Output:
	// 4 <nil>
}

func ExampleNamed() {
	withNamedWorkers := options.New(
		func(config *exampleConfig, value int) {
			config.workers = value
		},
		options.Named("workers", options.Min(1), options.Max(32)),
	)

	_, err := options.Apply(withNamedWorkers(0))
	fmt.Println(err)

	// Output:
	// workers: must be greater than or equal to 1: value=0
}

func ExampleCheck() {
	even := options.Check(func(value int, report options.Report) {
		if value%2 != 0 {
			report(options.ValidationError{Reason: "must be even"})
		}
	})
	withEvenWorkers := options.New(
		func(config *exampleConfig, value int) {
			config.workers = value
		},
		even,
	)

	config, err := options.Apply(withEvenWorkers(4))
	fmt.Println(config.workers, err)

	// Output:
	// 4 <nil>
}

func ExampleApplyTo() {
	defaults := exampleConfig{timeout: 30 * time.Second, workers: 2}
	config, err := options.ApplyTo(defaults, withWorkers(4))
	fmt.Println(config.timeout, config.workers, err)

	// Output:
	// 30s 4 <nil>
}
