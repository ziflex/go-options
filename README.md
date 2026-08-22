# go-options

A lightweight, generic functional options library for Go with built-in validation support.

## Overview

`go-options` provides a clean and type-safe way to implement the functional options pattern in Go. It leverages generics to work with any configuration struct and includes a built-in mechanism for reporting and collecting validation errors.

## Installation

```bash
go get github.com/ziflex/go-options
```

## API

### Types

- `Option[T any]`: A function type `func(*T, Report)` used to modify a configuration of type `T`.
- `Report`: A callback function `func(ValidationError)` used within an `Option` to report validation errors.
- `Validator[V any]`: A function type `func(V, Report)` used to validate an option value without receiving the destination configuration.
- `ValidationError`: A struct representing a validation failure, containing `Field`, `Value`, and `Reason`.

### Functions

- `Apply(opts...)` creates a zero-value configuration and applies the options.
- `ApplyTo(initial, opts...)` applies options to an existing configuration.
- `With(value, setter, validators...)` creates one option.
- `New(setter, validators...)` creates a reusable option constructor.
- `Check(check)` adapts custom validation logic.
- `Named(field, validators...)` adds optional diagnostic context.

Validation failures are collected with `errors.Join`. All validators run, but the
setter is called only when none of them reports a failure. Nil options and nil
validators are ignored.

`ApplyWithValues` remains available as a deprecated alias for `ApplyTo`.

## Examples

### Declaration styles

`New` creates concise reusable option constructors, while `With` supports
traditional function declarations. Both styles use the same validation and
assignment behavior and can be used together:

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

var WithTimeout = options.New(
	func(config *Config, value time.Duration) {
		config.Timeout = value
	},
	options.Min(time.Second),
)

func WithWorkers(value int) options.Option[Config] {
	return options.With(
		value,
		func(config *Config, value int) {
			config.Workers = value
		},
		options.Min(1),
		options.Max(32),
	)
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

### Named diagnostics

Validators do not require a field name. Use `Named` only when the additional
context is useful:

```go
var WithNamedWorkers = options.New(
	func(config *Config, value int) {
		config.Workers = value
	},
	options.Named(
		"workers",
		options.Min(1),
		options.Max(32),
	),
)
```

### Custom validation

`Check` is the escape hatch for application-specific rules. A check may report
more than one `ValidationError`:

```go
var Even = options.Check(func(value int, report options.Report) {
	if value%2 != 0 {
		report(options.ValidationError{
			Reason: "must be even",
			Value:  fmt.Sprint(value),
		})
	}
})

var WithEvenWorkers = options.New(
	func(config *Config, value int) {
		config.Workers = value
	},
	Even,
)
```

### Built-in validators

The package includes `NotNil`, `NotZero`, `NotEmpty`, `Min`, `Max`, `MinLen`,
`MaxLen`, and `OneOf`. `NotNil[V]` rejects nil pointers, interfaces (including
typed nils), slices, maps, functions, and channels; values of non-nilable types
always pass. String length is measured in bytes, matching Go's `len`.

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
