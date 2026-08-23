# go-options

A lightweight, generic functional options library for Go with built-in validation support.

## Overview

`go-options` provides a clean and type-safe way to implement the functional options pattern in Go. It leverages generics to work with any configuration struct and composes option and validation failures with ordinary Go errors.

## Installation

```bash
go get github.com/ziflex/go-options
```

## API

### Types

- `Option[T any]`: A function type `func(*T) error` used to modify a configuration of type `T`.
- `Builder[C, V any]`: A value-based builder that describes one option before producing it with `Build`.
- `Validator[V any]`: A function type `func(V) error` used to validate an option value without receiving the destination configuration.
- `ValidationError`: A struct representing a validation failure, containing `Field`, `Value`, and `Reason`.

### Functions

- `Apply(opts...)` creates a zero-value configuration and applies the options.
- `ApplyTo(initial, opts...)` applies options to an existing configuration.
- `New(setter)` creates an option builder and infers its configuration and value types from the setter.

`Apply` invokes every option and combines returned failures with `errors.Join`.
Builder-produced options run every validator, but call their setter only when all
validators return `nil`. Validators and custom options may return `errors.Join`
to describe multiple failures. Nil options and nil validators are ignored.

`ApplyWithValues` remains available as a deprecated alias for `ApplyTo`.

## Examples

### Building options

Option construction has five stages:

- `New` defines how the option modifies its configuration.
- `Value` binds the required option value, including an explicit zero value.
- `Named` optionally adds field context to validation failures.
- `Validators` optionally appends validators in execution order.
- `Build` produces the final `Option`.

Builder methods return updated values, so base builders can be reused safely.
Conventional `WithX` functions build one option per call:

```go
package main

import (
	"fmt"
	"time"

	"github.com/ziflex/go-options"
)

type Config struct {
	Timeout time.Duration
	Workers int
}

func WithWorkers(workers int) options.Option[Config] {
	return options.New(func(config *Config, value int) {
		config.Workers = value
	}).
		Value(workers).
		Validators(options.Min(1), options.Max(32)).
		Build()
}

func WithTimeout(timeout time.Duration) options.Option[Config] {
	return options.New(
		func(config *Config, value time.Duration) {
			config.Timeout = value
		},
	).
		Value(timeout).
		Named("timeout").
		Validators(options.Min(time.Second)).
		Build()
}

func main() {
	config, err := options.Apply(
		WithTimeout(5*time.Second),
		WithWorkers(4),
	)
	if err != nil {
		panic(err)
	}

	fmt.Println(config.Timeout, config.Workers)
}
```

### Custom options

Options may also be implemented directly. Return `nil` after a successful
mutation or return an ordinary Go error when the option cannot be applied:

```go
func WithWorkers(workers int) options.Option[Config] {
	return func(config *Config) error {
		if workers < 1 {
			return options.ValidationError{
				Field:  "workers",
				Reason: "must be positive",
			}
		}

		config.Workers = workers

		return nil
	}
}
```

A custom option with independent failures can return
`errors.Join(firstError, secondError)`.

### Named diagnostics

Validators do not require a field name. Use `Named` only when the additional
context is useful:

```go
func WithNamedWorkers(workers int) options.Option[Config] {
	return options.New(func(config *Config, value int) {
		config.Workers = value
	}).
		Value(workers).
		Named("workers").
		Validators(options.Min(1), options.Max(32)).
		Build()
}
```

### Custom validation

Use `Validator` for application-specific rules. Return `nil` when the value is
valid or an error describing why it is invalid:

```go
var Even = options.Validator[int](func(value int) error {
	if value%2 != 0 {
		return options.ValidationError{
			Reason: "must be even",
			Value:  fmt.Sprint(value),
		}
	}

	return nil
})

func WithEvenWorkers(workers int) options.Option[Config] {
	return options.New(func(config *Config, value int) {
		config.Workers = value
	}).
		Value(workers).
		Validators(Even).
		Build()
}
```

Return `errors.Join` when one validator needs to describe multiple failures:

```go
return errors.Join(
	options.ValidationError{Reason: "must be even"},
	options.ValidationError{Reason: "must be positive"},
)
```

### Built-in validators

The package includes `NotNil`, `NotNilPtr`, `NotZero`, `NotEmpty`, `Min`, `Max`,
`MinLen`, `MaxLen`, and `OneOf`. `NotNil[V]` is the general-purpose validator: it
rejects nil pointers, interfaces (including typed nils), slices, maps, functions,
and channels, while values of non-nilable types always pass. `NotNilPtr[T]` is the
reflection-free alternative for callers that want stronger pointer type intent.
String length is measured in bytes, matching Go's `len`.

Slices use `SliceNotEmpty`, `SliceMinLen`, and `SliceMaxLen`; maps use
`MapNotEmpty`, `MapMinLen`, and `MapMaxLen`. These helpers are statically typed so
unsupported kinds fail at compile time. The first type argument is sufficient for
named collection types:

```go
type Names []string
type Labels map[string]int

var NamesRequired = options.SliceNotEmpty[Names]()
var NamesLimited = options.SliceMaxLen[Names](10)
var LabelsRequired = options.MapNotEmpty[Labels]()
var LabelsLimited = options.MapMaxLen[Labels](10)
```

### Applying to an existing configuration

```go
defaults := Config{Timeout: 30 * time.Second}
config, err := options.ApplyTo(defaults, WithTimeout(5*time.Second))
```

## License

MIT
